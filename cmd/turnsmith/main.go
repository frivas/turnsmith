package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cli"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
)

const developmentVersion = "0.1.0-dev"

// Linker variables are build metadata, never generation state.
var (
	version string
	commit  string
	date    string
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
	info, _ := debug.ReadBuildInfo()
	build := resolveBuildMetadata(cli.Build{Version: version, Commit: commit, Date: date}, info)
	runner := cli.CLI{Service: app.Service{Registry: registry, Entropy: rand.Reader, Version: build.Version}, Build: build}
	return runner.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
