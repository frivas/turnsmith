package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/terminal"
	"io"
	"os"
	"strconv"
	"strings"
)

type commandOptions struct {
	values map[string]string
	seen   map[string]bool
	pos    []string
}

func options(args []string, defaults map[string]string, booleans ...string) (commandOptions, error) {
	o := commandOptions{values: map[string]string{}, seen: map[string]bool{}}
	bools := map[string]bool{"help": true}
	for _, b := range booleans {
		bools[b] = true
	}
	for k, v := range defaults {
		o.values[k] = v
	}
	o.values["help"] = "false"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			o.pos = append(o.pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			o.pos = append(o.pos, arg)
			continue
		}
		k, v, has := strings.Cut(arg, "=")
		switch k {
		case "-h":
			k = "--help"
		case "-p":
			k = "--puzzle"
		case "-l":
			k = "--length"
		case "-n":
			k = "--count"
		}
		k = strings.TrimPrefix(k, "--")
		if _, ok := o.values[k]; !ok {
			return o, app.Invalid("unknown flag %q", arg)
		}
		if o.seen[k] {
			return o, app.Invalid("--%s supplied more than once", k)
		}
		o.seen[k] = true
		if bools[k] {
			if !has {
				v = "true"
			}
			if v != "true" && v != "false" {
				return o, app.Invalid("--%s must be true or false", k)
			}
		} else if !has {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return o, app.Invalid("--%s requires a value", k)
			}
			i++
			v = args[i]
		}
		o.values[k] = v
	}
	return o, nil
}
func (o commandOptions) yes(key string) bool { return o.values[key] == "true" }
func (o commandOptions) number(key string, minValue, maxValue int) (int, error) {
	v := o.values[key]
	for _, r := range v {
		if r < '0' || r > '9' {
			return 0, app.Invalid("--%s must be an integer between %d and %d", key, minValue, maxValue)
		}
	}
	n, e := strconv.ParseUint(v, 10, 64)
	if v == "" || e != nil || n < uint64(minValue) || n > uint64(maxValue) {
		return 0, app.Invalid("--%s must be an integer between %d and %d", key, minValue, maxValue)
	}
	return int(n), nil
}
func invalid(err error) error {
	if err == nil {
		return nil
	}
	return app.Invalid("%s", err)
}
func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
func readBounded(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, app.Invalid("input is required")
	}
	data, e := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(data) > 1<<20 {
		return nil, app.Invalid("input exceeds 1 MiB")
	}
	return data, nil
}
func (c CLI) checkTerminal(w io.Writer) error {
	if !terminal.IsTerminal(c.Input) || !terminal.IsTerminal(w) || os.Getenv("TERM") == "dumb" {
		return app.Invalid("interactive mode requires terminal stdin and stdout; use inspect --steps or --diagram when piping")
	}
	return nil
}
func commandHelp(w io.Writer, name string) error {
	help := map[string]string{
		"inspect":    "turnsmith inspect --alg \"R U R' U'\" [--steps|--diagram|--json] [--puzzle 333] [--color auto|always|never] [--color-depth auto|16|256|truecolor]\n  Input: --alg TEXT, one quoted positional algorithm, --file PATH, or --stdin.\n  Supports Turnsmith JSON, saved inspection JSON, and daily practice sets; --index N selects a batch scramble.\n  Notation: outer/wide moves, M/E/S, x/y/z, groups, commutators and conjugates.\n",
		"inverse":    "turnsmith inverse --alg \"R U R' U'\" [--json]\n  Also accepts --file PATH or --stdin; --index N selects saved batch input.\n",
		"guide":      "turnsmith guide [--alg TEXT|--file PATH] [--puzzle 333] [--seed N] [--method METHOD] [--length N]\n  No input generates one scramble. Enter/right/space advances; left goes back with inverse guidance.\n  Home resets the displayed guide; End shows the final state; q exits. Requires terminal stdin/stdout.\n",
		"sessions":   "turnsmith sessions [list|new|show|add|edit|delete|delete-session|export|import|recover|lock|unlock] [NAME]\n  --data-dir PATH --json --session NAME\n  add: --time 12.345 --scramble \"R U\" [--penalty none|+2|DNF] [--notes TEXT]\n  edit/delete: --id ID; edit --time/--penalty/--notes\n  export/import: --format json|csv [--file PATH] [--replace]\n  recover restores a validated backup. unlock requires the exact lock token after stopping the owner.\n",
		"timer":      "turnsmith timer [--session NAME] [--data-dir PATH] [--puzzle 333] [--method METHOD] [--seed N] [--length N] [--inspection 0|15]\n  Portable press-based timing: Space/Enter starts or stops; n next; 2 +2; d DNF; 0 clear penalty; q exit.\n  Solves are saved locally only in this explicit mode.\n",
		"train":      "turnsmith train --file CASES.json [--mode balanced|learning|slowest|sequential] [--seed N] [--data-dir PATH]\n  --list / --json lists validated cases. --export-progress or --import-progress PATH backs up/restores progress.\n  Space starts/stops, then y/Space confirms success or d records unsuccessful; h hint; n next; l/k/0 learning/known/new.\n  Actions: list, status --case ID --status STATUS, export [--output PATH], import --input PATH [--replace], recover, lock, unlock --token TOKEN.\n",
		"daily":      "turnsmith daily [--date YYYY-MM-DD] [--name NAME] [--puzzle 333] [--method METHOD] [--count N] [--length N] [--json] [--output PATH]\n  UTC dates and versioned seed derivation produce a shareable practice set; name/count/length do not affect its seed.\n",
		"completion": "turnsmith completion bash|zsh|fish|powershell\n  Prints shell completion definitions; redirect or source them explicitly.\n",
	}
	_, err := io.WriteString(w, "Usage: "+help[name])
	return err
}
func (c CLI) extended(ctx context.Context, name string, args []string, w io.Writer) error {
	switch name {
	case "inspect", "inverse", "guide":
		return c.sequenceCommand(ctx, name, args, w)
	case "sessions":
		return c.sessionsCommand(ctx, args, w)
	case "timer":
		return c.timerCommand(ctx, args, w)
	case "train":
		return c.trainCommand(ctx, args, w)
	case "daily":
		return c.dailyCommand(ctx, args, w)
	case "completion":
		return c.completionCommand(args, w)
	}
	return app.Invalid("unknown command %q", name)
}
