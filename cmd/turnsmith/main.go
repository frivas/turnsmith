package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/frivas/turnsmith/internal/cube444"
	"github.com/frivas/turnsmith/internal/randomstate"
	"github.com/frivas/turnsmith/internal/terminal"
	"github.com/frivas/turnsmith/internal/view"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cli"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
)

const developmentVersion = "0.3.0-dev"

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
	reg333 := cube333.Registration()
	reg333.Generators = append(reg333.Generators, randomstate.Cube333{})
	registry, err := puzzle.New(reg333, randomstate.Registration222(), cube444.Registration())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: initialize registry: %s\n", err)
		return 1
	}
	info, _ := debug.ReadBuildInfo()
	build := resolveBuildMetadata(cli.Build{Version: version, Commit: commit, Date: date}, info)
	color := terminal.IsTerminal(os.Stdout) && os.Getenv("TERM") != "dumb" && os.Getenv("NO_COLOR") == ""
	depth := "16"
	if strings.Contains(os.Getenv("TERM"), "256color") {
		depth = "256"
	}
	if v := strings.ToLower(os.Getenv("COLORTERM")); v == "truecolor" || v == "24bit" {
		depth = "truecolor"
	}
	runner := cli.CLI{
		Service: app.Service{Registry: registry, Entropy: rand.Reader, Version: build.Version}, Build: build,
		Guides: map[string]cli.StepRenderer{"222": view.Guide{Size: 2}, "333": view.Guide{Size: 3}, "444": view.Guide{Size: 4}}, Color: color, Input: os.Stdin, ColorDepth: depth,
	}
	return runner.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
