// Package wcaengine talks to the separately licensed native Go scramble backend.
// The CLI shares no linked solver code with that executable.
package wcaengine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Message struct {
	Type             string   `json:"type"`
	Protocol         int      `json:"protocol,omitempty"`
	AlgorithmVersion string   `json:"algorithm_version,omitempty"`
	Word             string   `json:"word,omitempty"`
	Moves            []string `json:"moves,omitempty"`
	Error            string   `json:"error,omitempty"`
}

type cappedLog struct {
	mu   sync.Mutex
	data []byte
}

func (b *cappedLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	if len(b.data) < 16384 {
		b.data = append(b.data, p[:min(len(p), 16384-len(b.data))]...)
	}
	return n, nil
}
func (b *cappedLog) String() string { b.mu.Lock(); defer b.mu.Unlock(); return string(b.data) }

type Client struct {
	life    context.Context
	cancel  context.CancelFunc
	gate    chan struct{}
	path    string
	cmd     *exec.Cmd
	input   io.WriteCloser
	encoder *json.Encoder
	scanner *bufio.Scanner
	log     *cappedLog
}

func New(path string) *Client {
	life, cancel := context.WithCancel(context.Background())
	return &Client{gate: make(chan struct{}, 1), path: path, life: life, cancel: cancel}
}
func (c *Client) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "wca", AlgorithmVersion: "1", DirectEntropy: true, Description: "WCA-style random-state 4x4 via native Go backend", Guarantees: "Uniform reachable-state sampling with minimum-distance filtering; reaches sampled state", Limitations: "Requires the separate GPL-licensed turnsmith-wca executable; bounded initialization/search; practice, not official WCA approval"}
}
func backendPath() (string, error) {
	if p := os.Getenv("TURNSMITH_WCA_BACKEND"); p != "" {
		return p, nil
	}
	name := "turnsmith-wca"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if exe, e := os.Executable(); e == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if info, e := os.Stat(p); e == nil && !info.IsDir() {
			return p, nil
		}
	}
	if p, e := exec.LookPath(name); e == nil {
		return p, nil
	}
	return "", fmt.Errorf("native 4x4 backend missing: install turnsmith-wca alongside turnsmith from the release archive, or set TURNSMITH_WCA_BACKEND")
}
func (c *Client) stop() {
	if c.input != nil {
		_ = c.input.Close()
	}
	if c.cmd != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	c.cmd = nil
	c.input = nil
	c.encoder = nil
	c.scanner = nil
}
func (c *Client) Close() error {
	c.cancel()
	c.gate <- struct{}{}
	defer func() { <-c.gate }()
	c.stop()
	return nil
}
func (c *Client) read() (Message, error) {
	var m Message
	if !c.scanner.Scan() {
		e := c.scanner.Err()
		if e == nil {
			e = io.ErrUnexpectedEOF
		}
		// Wait drains exec's stderr copier before we snapshot its diagnostics.
		// Stdout EOF alone does not mean the final stderr write has arrived.
		c.stop()
		return m, fmt.Errorf("read native backend: %w; %s", e, c.log.String())
	}
	if e := json.Unmarshal(c.scanner.Bytes(), &m); e != nil {
		return m, fmt.Errorf("decode native backend: %w", e)
	}
	return m, nil
}
func (c *Client) start() error {
	path := c.path
	var e error
	if path == "" {
		path, e = backendPath()
		if e != nil {
			return e
		}
	}
	c.cmd = exec.Command(path, "--stdio")
	c.log = &cappedLog{}
	c.cmd.Stderr = c.log
	c.input, e = c.cmd.StdinPipe()
	if e != nil {
		c.cmd = nil
		return e
	}
	output, e := c.cmd.StdoutPipe()
	if e != nil {
		_ = c.input.Close()
		c.cmd = nil
		return e
	}
	if e = c.cmd.Start(); e != nil {
		_ = c.input.Close()
		c.cmd = nil
		return fmt.Errorf("start native 4x4 backend: %w", e)
	}
	c.encoder = json.NewEncoder(c.input)
	c.scanner = bufio.NewScanner(output)
	c.scanner.Buffer(make([]byte, 4096), 1<<20)
	return nil
}
func (c *Client) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (result puzzle.Generated, err error) {
	if options.Length != nil {
		return result, fmt.Errorf("wca does not accept a fixed length")
	}
	if rng == nil {
		return result, fmt.Errorf("random source is required")
	}
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return result, ctx.Err()
	case <-c.life.Done():
		return result, fmt.Errorf("native backend client is closed")
	}
	defer func() { <-c.gate }()
	if e := ctx.Err(); e != nil {
		return result, e
	}
	if c.life.Err() != nil {
		return result, fmt.Errorf("native backend client is closed")
	}
	fresh := c.cmd == nil
	if fresh {
		if e := c.start(); e != nil {
			return result, e
		}
	}
	process := c.cmd.Process
	stopWatch := context.AfterFunc(ctx, func() { _ = process.Kill() })
	defer stopWatch()
	stopClose := context.AfterFunc(c.life, func() { _ = process.Kill() })
	defer stopClose()
	defer func() {
		if err != nil {
			c.stop()
			if ctx.Err() != nil {
				err = ctx.Err()
			}
		}
	}()
	if fresh {
		hello, e := c.read()
		if e != nil {
			return result, e
		}
		if hello.Type != "ready" || hello.Protocol != 1 || hello.AlgorithmVersion != "1" {
			return result, fmt.Errorf("unsupported native backend protocol")
		}
	}
	if e := c.encoder.Encode(Message{Type: "generate"}); e != nil {
		return result, e
	}
	for draws := 0; draws < 100000; draws++ {
		if e := ctx.Err(); e != nil {
			return result, e
		}
		m, e := c.read()
		if e != nil {
			return result, e
		}
		switch m.Type {
		case "random":
			word := rng.Uint64()
			if e := ctx.Err(); e != nil {
				return result, e
			}
			if e := c.encoder.Encode(Message{Type: "word", Word: strconv.FormatUint(word, 10)}); e != nil {
				return result, e
			}
		case "error":
			return result, fmt.Errorf("native 4x4 backend: %s", m.Error)
		case "result":
			if len(m.Moves) == 0 || len(m.Moves) > 200 {
				return result, fmt.Errorf("invalid native backend move count")
			}
			state, _ := cubestate.New(4)
			for _, token := range m.Moves {
				moves, e := alg.Parse(token, 1)
				if e != nil {
					return result, fmt.Errorf("invalid backend token: %w", e)
				}
				if len(moves) != 1 || moves[0].String() != token || !strings.ContainsRune("UDLRFB", rune(moves[0].Face)) || moves[0].Width > 2 {
					return result, fmt.Errorf("invalid native 4x4 notation %q", token)
				}
				if e = state.Apply(moves[0]); e != nil {
					return result, e
				}
			}
			return puzzle.Generated{Moves: append([]string(nil), m.Moves...)}, nil
		default:
			return result, fmt.Errorf("unexpected native backend message %q", m.Type)
		}
	}
	return result, fmt.Errorf("native backend exceeded entropy request limit")
}
