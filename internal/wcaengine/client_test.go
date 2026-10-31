package wcaengine

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The same test executable is a native protocol peer when the broker launches
// it with --stdio. No shell, Java runtime, or real solver tables are involved.
func TestMain(m *testing.M) {
	if os.Getenv("TURNSMITH_BROKER_TEST_MODE") != "" && len(os.Args) > 1 && os.Args[1] == "--stdio" {
		brokerTestPeer()
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func brokerTestPeer() {
	mode := os.Getenv("TURNSMITH_BROKER_TEST_MODE")
	marker := os.Getenv("TURNSMITH_BROKER_TEST_MARKER")
	if marker != "" {
		f, e := os.OpenFile(marker, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e == nil {
			_, _ = fmt.Fprintln(f, "start")
			_ = f.Close()
		}
	}
	enc := json.NewEncoder(os.Stdout)
	dec := json.NewDecoder(bufio.NewReader(os.Stdin))
	if mode == "block-hello" {
		time.Sleep(time.Hour)
		return
	}
	if mode == "child-failure" {
		fmt.Fprintln(os.Stderr, "fixture child failure")
		os.Exit(7)
	}
	if mode == "bad-handshake" {
		_ = enc.Encode(Message{Type: "ready", Protocol: 99})
		return
	}
	if mode == "oversized" {
		_, _ = io.WriteString(os.Stdout, strings.Repeat("x", (1<<20)+1)+"\n")
		return
	}
	version := "1"
	if mode == "bad-algorithm-version" {
		version = "2"
	}
	if mode == "missing-algorithm-version" {
		version = ""
	}
	_ = enc.Encode(map[string]any{"type": "ready", "protocol": 1, "algorithm_version": version})
	for {
		var request Message
		if dec.Decode(&request) != nil {
			return
		}
		if request.Type != "generate" {
			os.Exit(8)
		}
		switch mode {
		case "block-result":
			if marker != "" {
				_ = os.WriteFile(marker+".blocked", []byte("blocked"), 0600)
			}
			time.Sleep(time.Hour)
			return
		case "block-write":
			if marker != "" {
				_ = os.WriteFile(marker+".blocked", []byte("blocked"), 0600)
			}
			for i := 0; i < 100000; i++ {
				if enc.Encode(Message{Type: "random"}) != nil {
					return
				}
			}
			time.Sleep(time.Hour)
			return
		case "bad-json":
			_, _ = io.WriteString(os.Stdout, "{not-json}\n")
			return
		case "unknown-message":
			_ = enc.Encode(Message{Type: "unexpected"})
			return
		case "reported-error":
			_ = enc.Encode(Message{Type: "error", Error: "fixture search bound"})
			return
		case "empty-result":
			_ = enc.Encode(Message{Type: "result", Moves: []string{}})
			return
		case "oversized-result":
			moves := make([]string, 201)
			for i := range moves {
				moves[i] = "R"
			}
			_ = enc.Encode(Message{Type: "result", Moves: moves})
			return
		case "invalid-token", "rotation-token", "decorated-token":
			token := "Q"
			if mode == "rotation-token" {
				token = "x"
			}
			if mode == "decorated-token" {
				token = "(R)"
			}
			_ = enc.Encode(Message{Type: "result", Moves: []string{token}})
			return
		}
		for _, want := range []string{"18446744073709551615", "9007199254740993", "42"} {
			_ = enc.Encode(Message{Type: "random"})
			var word Message
			if dec.Decode(&word) != nil {
				return
			}
			if word.Type != "word" || word.Word != want {
				_ = enc.Encode(Message{Type: "error", Error: fmt.Sprintf("word precision: got %q want %q", word.Word, want)})
				return
			}
		}
		_ = enc.Encode(Message{Type: "result", Moves: []string{"R", "Uw2", "F'"}})
	}
}

type wordFixture struct{ index int }

func (s *wordFixture) Uint64() uint64 {
	words := []uint64{^uint64(0), 9007199254740993, 42}
	v := words[s.index%len(words)]
	s.index++
	return v
}
func brokerClient(t *testing.T, mode string) (*Client, string) {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "processes")
	t.Setenv("TURNSMITH_BROKER_TEST_MODE", mode)
	t.Setenv("TURNSMITH_BROKER_TEST_MARKER", marker)
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	client := New(exe)
	t.Cleanup(func() { _ = client.Close() })
	return client, marker
}
func waitMarker(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, e := os.Stat(path); e == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("child did not reach blocked protocol stage")
}
func TestNativeHandshakePrecisionAndPersistentBatch(t *testing.T) {
	c, marker := brokerClient(t, "normal")
	for i := 0; i < 3; i++ {
		result, e := c.Generate(context.Background(), &wordFixture{}, puzzle.Options{})
		if e != nil || strings.Join(result.Moves, " ") != "R Uw2 F'" {
			t.Fatal(result, e)
		}
	}
	data, e := os.ReadFile(marker)
	if e != nil || strings.Count(string(data), "start") != 1 {
		t.Fatal("batch restarted native process", string(data), e)
	}
	if e = c.Close(); e != nil {
		t.Fatal(e)
	}
	if e = c.Close(); e != nil {
		t.Fatal("close not idempotent", e)
	}
}
func TestNativeRejectsMalformedResponses(t *testing.T) {
	for _, mode := range []string{"bad-handshake", "bad-algorithm-version", "missing-algorithm-version", "oversized", "child-failure", "bad-json", "unknown-message", "reported-error", "empty-result", "oversized-result", "invalid-token", "rotation-token", "decorated-token"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := brokerClient(t, mode)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			got, e := c.Generate(ctx, &wordFixture{}, puzzle.Options{})
			if e == nil || len(got.Moves) != 0 {
				t.Fatal("invalid peer response accepted", mode, got, e)
			}
			if mode == "child-failure" && !strings.Contains(e.Error(), "fixture child failure") {
				t.Fatal("stderr evidence missing", e)
			}
			if mode == "oversized" && !strings.Contains(e.Error(), "token too long") {
				t.Fatal("unbounded read", e)
			}
		})
	}
}
func TestNativeCancellationKillsBlockedIO(t *testing.T) {
	for _, mode := range []string{"block-hello", "block-result", "block-write"} {
		t.Run(mode, func(t *testing.T) {
			c, marker := brokerClient(t, mode)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { _, e := c.Generate(ctx, &wordFixture{}, puzzle.Options{}); done <- e }()
			if mode != "block-hello" {
				marker += ".blocked"
			}
			waitMarker(t, marker)
			if mode == "block-write" {
				time.Sleep(30 * time.Millisecond)
			}
			cancel()
			select {
			case e := <-done:
				if !errors.Is(e, context.Canceled) {
					t.Fatal(e)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("cancellation left blocked child I/O")
			}
			closed := make(chan struct{})
			go func() { _ = c.Close(); close(closed) }()
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Fatal("close blocked after cancellation")
			}
		})
	}
}
func TestNativeWaitingGateCancellation(t *testing.T) {
	c, marker := brokerClient(t, "block-result")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() { _, e := c.Generate(ctx, &wordFixture{}, puzzle.Options{}); first <- e }()
	waitMarker(t, marker+".blocked")
	waiting, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer stop()
	_, e := c.Generate(waiting, &wordFixture{}, puzzle.Options{})
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	cancel()
	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("first request stuck")
	}
}
func TestNativeCloseInterruptsActiveRequest(t *testing.T) {
	c, marker := brokerClient(t, "block-result")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := c.Generate(ctx, &wordFixture{}, puzzle.Options{}); done <- e }()
	waitMarker(t, marker+".blocked")
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case e := <-closed:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(300 * time.Millisecond):
		cancel()
		<-done
		<-closed
		t.Fatal("Close waited indefinitely for an active child instead of interrupting it")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close left Generate blocked")
	}
}
func TestNativeMissingBackendAndInputValidation(t *testing.T) {
	t.Setenv("TURNSMITH_WCA_BACKEND", "")
	t.Setenv("PATH", t.TempDir())
	c := New("")
	defer c.Close()
	_, e := c.Generate(context.Background(), &wordFixture{}, puzzle.Options{})
	if e == nil || !strings.Contains(e.Error(), "backend missing") {
		t.Fatal(e)
	}
	n := 1
	for _, tc := range []struct {
		rng     random.Source
		options puzzle.Options
	}{{nil, puzzle.Options{}}, {&wordFixture{}, puzzle.Options{Length: &n}}} {
		if _, e = c.Generate(context.Background(), tc.rng, tc.options); e == nil {
			t.Fatal("invalid request accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = c.Generate(ctx, &wordFixture{}, puzzle.Options{}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
func TestNativeStderrLogBound(t *testing.T) {
	var log cappedLog
	data := []byte(strings.Repeat("x", 100000))
	n, e := log.Write(data)
	if e != nil || n != len(data) || len(log.String()) != 16384 {
		t.Fatal(n, e, len(log.String()))
	}
}

func TestNativeRestartsAfterProtocolFailure(t *testing.T) {
	c, marker := brokerClient(t, "unknown-message")
	if _, e := c.Generate(context.Background(), &wordFixture{}, puzzle.Options{}); e == nil {
		t.Fatal("invalid protocol accepted")
	}
	t.Setenv("TURNSMITH_BROKER_TEST_MODE", "normal")
	if _, e := c.Generate(context.Background(), &wordFixture{}, puzzle.Options{}); e != nil {
		t.Fatal("failed process poisoned subsequent request", e)
	}
	data, e := os.ReadFile(marker)
	if e != nil || strings.Count(string(data), "start") != 2 {
		t.Fatal(string(data), e)
	}
}
