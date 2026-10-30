// Package cli owns parsing, presentation selection, and process status mapping.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"scramble/internal/app"
	"scramble/internal/output"
	"scramble/internal/puzzle"
	"scramble/internal/random"
)

type Build struct{ Version, Commit, Date string }
type CLI struct {
	Service app.Service
	Build   Build
}

const help = `scramble generates filtered random-move scrambles for casual practice.

Usage:
  scramble [flags]
  scramble puzzles [--json]
  scramble version
  scramble help

Generation flags:
  -p, --puzzle ID    Puzzle ID or alias (default 333)
      --method ID    Generation method (default random-move for 333)
  -l, --length N     Tokens per scramble (default 25; range 1..1000)
  -n, --count N      Scrambles (default 1; range 1..10000)
      --seed N       Unsigned decimal 64-bit root seed (including zero)
      --json         Emit a versioned JSON document
  -h, --help         Show help
      --version      Print name and version

At most 1000000 total moves may be requested.
Notation: U D L R F B, optionally followed by 2 or an ASCII apostrophe.
A half turn counts as one token. Random moves do not uniformly sample states
and are not approved for official competition. No runtime network access.
`
const puzzleHelp = "Usage: scramble puzzles [--json]\n\nList supported puzzles, aliases, methods, and capabilities.\n"

type parsed struct {
	command             string
	request             app.Request
	json, help, version bool
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
	p := parsed{command: "generate", request: app.Request{Puzzle: "333", Count: 1}}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "puzzles", "version", "help":
			p.command = args[0]
			args = args[1:]
		default:
			return p, app.Invalid("unexpected command or positional argument %q; run \"scramble help\"", args[0])
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
		allowed := key == "--help" || (p.command == "puzzles" && key == "--json") || (p.command == "generate" && (key == "--puzzle" || key == "--method" || key == "--length" || key == "--count" || key == "--seed" || key == "--json" || key == "--version"))
		if !allowed {
			return p, app.Invalid("unknown flag %q for %s; run \"scramble help\"", key, p.command)
		}
		if seen[key] {
			return p, app.Invalid("flag %s supplied more than once", key)
		}
		seen[key] = true
		boolean := key == "--json" || key == "--help" || key == "--version"
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
		}
	}
	if p.version && (p.help || len(seen) > 1) {
		return p, app.Invalid("--version cannot be combined with other flags")
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
		return write("scramble " + c.Build.Version + "\n")
	}
	if p.command == "version" {
		return write(fmt.Sprintf("scramble %s\ncommit: %s\nbuilt: %s\n", c.Build.Version, c.Build.Commit, c.Build.Date))
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
		result, e := c.Service.Generate(ctx, p.request)
		if e != nil {
			return e
		}
		if p.json {
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
