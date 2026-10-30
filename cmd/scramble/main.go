package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"

	"scramble/internal/app"
	"scramble/internal/cli"
	"scramble/internal/cube333"
	"scramble/internal/puzzle"
)

// Linker variables are build metadata, never generation state.
var (
	version = "0.1.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() { os.Exit(run()) }
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	registry, err := puzzle.New(cube333.Registration())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: initialize registry: %s\n", err)
		return 1
	}
	runner := cli.CLI{Service: app.Service{Registry: registry, Entropy: rand.Reader, Version: version}, Build: cli.Build{Version: version, Commit: commit, Date: date}}
	return runner.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
