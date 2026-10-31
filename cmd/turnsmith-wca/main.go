// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Francisco Rivas and contributors.
// Native standalone backend using the Go adaptation of Shuang Chen's solver.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/frivas/turnsmith/internal/randomstate"
	"github.com/frivas/turnsmith/internal/tnoodle444"
	"io"
	"math"
	"os"
	"os/signal"
	"strings"
)

var version = "0.3.0-dev"
var commit = "unknown"
var date = "unknown"

type message struct {
	Type             string   `json:"type"`
	Protocol         int      `json:"protocol,omitempty"`
	AlgorithmVersion string   `json:"algorithm_version,omitempty"`
	Word             string   `json:"word,omitempty"`
	Moves            []string `json:"moves,omitempty"`
	Error            string   `json:"error,omitempty"`
}
type pipeRandom struct {
	scan   *bufio.Scanner
	encode *json.Encoder
	cancel context.CancelFunc
	err    error
}

func (p *pipeRandom) fail(e error) uint64 { p.err = e; p.cancel(); return math.MaxUint64 }
func (p *pipeRandom) Uint64() uint64 {
	if p.err != nil {
		return math.MaxUint64
	}
	if e := p.encode.Encode(message{Type: "random"}); e != nil {
		return p.fail(e)
	}
	if !p.scan.Scan() {
		e := p.scan.Err()
		if e == nil {
			e = io.ErrUnexpectedEOF
		}
		return p.fail(e)
	}
	var m message
	if e := json.Unmarshal(p.scan.Bytes(), &m); e != nil {
		return p.fail(e)
	}
	if m.Type != "word" {
		return p.fail(fmt.Errorf("expected entropy word"))
	}
	n, e := random.ParseSeed(m.Word)
	if e != nil {
		return p.fail(e)
	}
	return n
}
func serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1<<20)
	enc := json.NewEncoder(out)
	if e := enc.Encode(message{Type: "ready", Protocol: 1, AlgorithmVersion: "1"}); e != nil {
		return e
	}
	for scan.Scan() {
		if e := ctx.Err(); e != nil {
			return e
		}
		var m message
		if e := json.Unmarshal(scan.Bytes(), &m); e != nil {
			return e
		}
		if m.Type != "generate" {
			return fmt.Errorf("expected generate request")
		}
		child, cancel := context.WithCancel(ctx)
		rng := &pipeRandom{scan: scan, encode: enc, cancel: cancel}
		moves, e := tnoodle444.Generate(child, rng, randomstate.Solve333Facelets)
		cancel()
		if rng.err != nil {
			return rng.err
		}
		if e != nil {
			if e := enc.Encode(message{Type: "error", Error: e.Error()}); e != nil {
				return e
			}
			continue
		}
		if e := enc.Encode(message{Type: "result", Moves: moves}); e != nil {
			return e
		}
	}
	return scan.Err()
}
func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("turnsmith-wca", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	stdio := flags.Bool("stdio", false, "serve the bounded JSON-line entropy protocol")
	showVersion := flags.Bool("version", false, "show backend version")
	license := flags.Bool("license", false, "show license and source information")
	seedText := flags.String("seed", "", "optional unsigned seed for reproducible practice")
	count := flags.Int("count", 1, "4x4 scrambles to generate, 1..1000")
	if e := flags.Parse(args); e != nil {
		return e
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if *showVersion {
		_, e := fmt.Fprintf(out, "turnsmith-wca %s\ncommit: %s\ndate: %s\n", version, commit, date)
		return e
	}
	if *license {
		_, e := io.WriteString(out, "turnsmith-wca uses GPL-3.0-or-later.\nCopyright (C) 2014 Shuang Chen; Go adaptation Copyright (C) 2026 Francisco Rivas and contributors.\nSource: https://github.com/frivas/turnsmith/tree/"+commit+"/internal/tnoodle444\nLicense: https://www.gnu.org/licenses/gpl-3.0.html\nThis program comes with absolutely no warranty.\n")
		return e
	}
	if *stdio {
		return serve(ctx, os.Stdin, out)
	}
	if *count < 1 || *count > 1000 {
		return fmt.Errorf("--count must be 1..1000")
	}
	var root uint64
	var e error
	if *seedText != "" {
		root, e = random.ParseSeed(*seedText)
		if e != nil {
			return e
		}
	}
	// Complete a standalone batch before emitting it, just as the main CLI does.
	var text strings.Builder
	for i := 0; i < *count; i++ {
		child, cancel := context.WithCancel(ctx)
		var source random.Source
		var entropy *random.EntropySource
		if *seedText != "" {
			source = random.Derive(root, "444", "wca", "1", uint64(i))
		} else {
			entropy = random.NewEntropySource(rand.Reader, func(error) { cancel() })
			source = entropy
		}
		moves, e := tnoodle444.Generate(child, source, randomstate.Solve333Facelets)
		cancel()
		if entropy != nil && entropy.Err() != nil {
			return entropy.Err()
		}
		if e != nil {
			return e
		}
		text.WriteString(strings.Join(moves, " ") + "\n")
	}
	_, e = io.WriteString(out, text.String())
	return e
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if e := run(ctx, os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "error:", e)
		os.Exit(1)
	}
}
