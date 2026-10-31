// Package cli owns parsing, presentation selection, and process status mapping.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/output"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
)

type Build struct{ Version, Commit, Date string }
type StepRenderer interface {
	Render(context.Context, io.Writer, []string, bool) error
}

type CLI struct {
	Service    app.Service
	Build      Build
	Guides     map[string]StepRenderer
	Color      bool
	Input      io.Reader
	ColorDepth string
}

const help = `turnsmith provides WCA-style cube practice and local terminal sessions.

Usage:
  turnsmith [flags]
  turnsmith puzzles [--json]
  turnsmith inspect|inverse|guide [flags]
  turnsmith sessions|timer|train|daily [flags]
  turnsmith completion bash|zsh|fish|powershell
  turnsmith version
  turnsmith help

Generation flags:
  -p, --puzzle ID    Puzzle ID or alias (default 333)
      --method ID    Generation method (default wca)
  -l, --length N     Tokens for random-move only (333:25, 444:40; range 1..1000)
  -n, --count N      Scrambles (default 1; range 1..10000)
      --seed N       Optional unsigned seed for reproducible practice, including zero
      --json         Emit a versioned JSON document
      --steps        Show a solved cube and each move as a tile diagram
      --diagram      Show only the final cube state
      --color-depth N Color depth: auto, 16, 256, truecolor
      --color MODE   Diagram colors: auto, always, never (default auto)
  -h, --help         Show help
      --version      Print name and version

At most 1000000 total moves may be requested.
Step diagrams support at most 1000 total moves and cannot be combined with JSON.
Run a command with --help for its options. 222 supports wca and random-state;
333 supports wca, random-move and random-state; 444 supports wca and random-move.
Notation includes outer turns and wide turns for 444.
Unseeded wca uses fresh cryptographic randomness; seeds select reproducible practice.
A half turn counts as one token. Random moves do not uniformly sample states
and are not approved for official competition. No runtime network access.
`
const puzzleHelp = "Usage: turnsmith puzzles [--json]\n\nList supported puzzles, aliases, methods, and capabilities.\n"

type parsed struct {
	command             string
	request             app.Request
	json, help, version bool
	steps, diagram      bool
	color, depth        string
}

func integer(value, flag string, max int) (int, error) {
	message := "--" + flag + " must be an integer between 1 and " + strconv.Itoa(max)
	if value == "" {
		return 0, app.Invalid("%s", message)
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, app.Invalid("%s", message)
		}
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil || n < 1 || n > uint64(max) {
		return 0, app.Invalid("%s", message)
	}
	return int(n), nil
}
func parse(args []string) (parsed, error) {
	p := parsed{command: "generate", request: app.Request{Puzzle: "333", Count: 1}, color: "auto", depth: "auto"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "puzzles", "version", "help":
			p.command = args[0]
			args = args[1:]
		default:
			return p, app.Invalid("unexpected command or positional argument %q; run \"turnsmith help\"", args[0])
		}
	}
	seen := make(map[string]bool)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
			return p, app.Invalid("unexpected positional argument %q", arg)
		}
		key, value, hasValue := strings.Cut(arg, "=")
		switch key {
		case "-p":
			key = "--puzzle"
		case "-l":
			key = "--length"
		case "-n":
			key = "--count"
		case "-h":
			key = "--help"
		}
		allowed := key == "--help" || (p.command == "puzzles" && key == "--json") || (p.command == "generate" && (key == "--puzzle" || key == "--method" || key == "--length" || key == "--count" || key == "--seed" || key == "--json" || key == "--version" || key == "--steps" || key == "--diagram" || key == "--color-depth" || key == "--color"))
		if !allowed {
			return p, app.Invalid("unknown flag %q for %s; run \"turnsmith help\"", key, p.command)
		}
		if seen[key] {
			return p, app.Invalid("flag %s supplied more than once", key)
		}
		seen[key] = true
		boolean := key == "--json" || key == "--help" || key == "--version" || key == "--steps" || key == "--diagram"
		if boolean {
			b := true
			if hasValue {
				if value != "true" && value != "false" {
					return p, app.Invalid("%s must be true or false", key)
				}
				b = value == "true"
			}
			switch key {
			case "--json":
				p.json = b
			case "--help":
				p.help = b
			case "--version":
				p.version = b
			case "--steps":
				p.steps = b
			case "--diagram":
				p.diagram = b
			}
			continue
		}
		if !hasValue {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return p, app.Invalid("%s requires a value", key)
			}
			i++
			value = args[i]
		}
		switch key {
		case "--puzzle":
			if value == "" {
				return p, app.Invalid("--puzzle requires a nonempty value")
			}
			p.request.Puzzle = value
		case "--method":
			if value == "" {
				return p, app.Invalid("--method requires a nonempty value")
			}
			p.request.Method = value
		case "--length":
			n, err := integer(value, "length", 1000)
			if err != nil {
				return p, err
			}
			p.request.Length = &n
		case "--count":
			n, err := integer(value, "count", 10000)
			if err != nil {
				return p, err
			}
			p.request.Count = n
		case "--seed":
			n, err := random.ParseSeed(value)
			if err != nil {
				return p, app.Invalid("%s", err)
			}
			p.request.Seed = &n
		case "--color-depth":
			if value != "auto" && value != "16" && value != "256" && value != "truecolor" {
				return p, app.Invalid("--color-depth must be auto, 16, 256, or truecolor")
			}
			p.depth = value
		case "--color":
			if value != "auto" && value != "always" && value != "never" {
				return p, app.Invalid("--color must be auto, always, or never")
			}
			p.color = value
		}
	}
	if p.version && (p.help || len(seen) > 1) {
		return p, app.Invalid("--version cannot be combined with other flags")
	}
	if p.steps && p.diagram {
		return p, app.Invalid("choose --steps or --diagram")
	}
	if (p.steps || p.diagram) && p.json {
		return p, app.Invalid("--steps cannot be combined with --json")
	}
	if (seen["--color"] || seen["--color-depth"]) && !p.steps && !p.diagram {
		return p, app.Invalid("--color and --color-depth require --steps or --diagram")
	}
	return p, nil
}
func (c CLI) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	err := c.run(ctx, args, stdout)
	if err == nil {
		return 0
	}
	code := 1
	var invalid *app.InvalidInput
	if errors.As(err, &invalid) {
		code = 2
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = 130
	}
	fmt.Fprintf(stderr, "error: %s\n", err)
	return code
}
func (c CLI) run(ctx context.Context, args []string, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(args) > 0 {
		switch args[0] {
		case "inspect", "inverse", "guide", "sessions", "timer", "train", "daily", "completion":
			return c.extended(ctx, args[0], args[1:], w)
		}
		if args[0] == "help" && len(args) == 2 {
			switch args[1] {
			case "inspect", "inverse", "guide", "sessions", "timer", "train", "daily", "completion":
				return commandHelp(w, args[1])
			}
		}
	}
	p, err := parse(args)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	write := func(s string) error {
		_, err := io.WriteString(w, s)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	}
	if p.command == "help" || p.help {
		if p.command == "puzzles" {
			return write(puzzleHelp)
		}
		return write(help)
	}
	if p.version {
		return write("turnsmith " + c.Build.Version + "\n")
	}
	if p.command == "version" {
		return write(fmt.Sprintf("turnsmith %s\ncommit: %s\ndate: %s\n", c.Build.Version, c.Build.Commit, c.Build.Date))
	}
	if p.command == "puzzles" {
		if p.json {
			err = output.JSON(w, struct {
				SchemaVersion int                 `json:"schema_version"`
				Puzzles       []puzzle.Descriptor `json:"puzzles"`
			}{1, c.Service.Registry.List()})
		} else {
			err = output.Puzzles(w, c.Service.Registry.List())
		}
	} else {
		var guide StepRenderer
		if p.steps || p.diagram {
			descriptor, ok := c.Service.Registry.Resolve(p.request.Puzzle)
			if !ok {
				return app.Invalid("unsupported puzzle %q; run \"turnsmith puzzles\" to list supported puzzles", p.request.Puzzle)
			}
			guide = c.Guides[descriptor.ID]
			if guide == nil {
				return app.Invalid("--steps is not available for puzzle %q", descriptor.ID)
			}
		}
		result, e := c.Service.Generate(ctx, p.request)
		if e != nil {
			return e
		}
		if p.steps || p.diagram {
			total := 0
			for _, scramble := range result.Scrambles {
				total += scramble.MoveCount
			}
			if total > 1000 {
				return app.Invalid("--steps supports at most 1000 total moves; reduce --length or --count")
			}
			color := p.color == "always" || (p.color == "auto" && c.Color)
			for _, scramble := range result.Scrambles {
				if err = write(fmt.Sprintf("Scramble %d/%d | seed %s\n%s\n\n", scramble.Index, len(result.Scrambles), result.Seed, scramble.Sequence)); err != nil {
					return err
				}
				if rich, ok := guide.(interface {
					RenderView(context.Context, io.Writer, []string, bool, string, bool) error
				}); ok {
					depth := p.depth
					if depth == "auto" {
						depth = c.ColorDepth
					}
					err = rich.RenderView(ctx, w, scramble.Moves, color, depth, p.diagram)
				} else if p.diagram {
					return app.Invalid("final diagrams unavailable for this puzzle")
				} else {
					err = guide.Render(ctx, w, scramble.Moves, color)
				}
				if err != nil {
					return fmt.Errorf("write step diagrams: %w", err)
				}
			}
		} else if p.json {
			err = output.JSON(w, result)
		} else {
			err = output.Text(w, result)
		}
	}
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
