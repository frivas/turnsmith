package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/daily"
)

func (c CLI) dailyCommand(ctx context.Context, args []string, out io.Writer) error {
	o, err := options(args, map[string]string{"date": "", "name": "", "puzzle": "333", "method": "", "count": "5", "length": "", "json": "false", "output": ""}, "json")
	if err != nil {
		return err
	}
	if o.yes("help") {
		return commandHelp(out, "daily")
	}
	if len(o.pos) != 0 {
		return app.Invalid("daily does not accept positional arguments; use --name NAME")
	}
	for _, key := range []string{"date", "name", "puzzle", "method", "output"} {
		if o.seen[key] && strings.TrimSpace(o.values[key]) == "" {
			return app.Invalid("--%s requires a nonempty value", key)
		}
	}
	name := o.values["name"]
	if len(name) > 256 || !utf8.ValidString(name) || strings.ContainsFunc(name, unicode.IsControl) {
		return app.Invalid("--name must be valid text of at most 256 bytes without control characters")
	}
	count, err := o.number("count", 1, 10000)
	if err != nil {
		return err
	}
	request := daily.Options{Date: o.values["date"], Name: name, Puzzle: o.values["puzzle"], Method: o.values["method"], Count: count}
	if o.seen["length"] {
		n, e := o.number("length", 1, 1000)
		if e != nil {
			return e
		}
		request.Length = &n
	}
	manifest, err := daily.Build(ctx, c.Service, request)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode practice set: %w", err)
	}
	encoded = append(encoded, '\n')
	// Saved sets share the inspection reader's bound, so every manifest emitted
	// here remains directly loadable by inspect and guide.
	if len(encoded) > 1<<20 {
		return app.Invalid("daily practice set exceeds 1 MiB; reduce --count or --length")
	}
	var rendered bytes.Buffer
	if o.yes("json") {
		rendered.Write(encoded)
	} else {
		fmt.Fprintf(&rendered, "%s\nDate: %s (%s) | seed: %s | seed policy: %s\nPuzzle: %s | method: %s | algorithm: %s | Turnsmith: %s\n\n", manifest.Name, manifest.Date, manifest.DatePolicy, manifest.Result.Seed, manifest.SeedPolicyVersion, manifest.Result.Puzzle, manifest.Result.Method, manifest.Result.AlgorithmVersion, manifest.Result.AppVersion)
		for _, scramble := range manifest.Result.Scrambles {
			fmt.Fprintf(&rendered, "%d. %s\n", scramble.Index, scramble.Sequence)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if o.seen["output"] {
		if err := saveDailyManifest(ctx, o.values["output"], encoded); err != nil {
			return err
		}
	}
	if _, err := out.Write(rendered.Bytes()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// Generate and encode before touching the destination. A sibling temporary file
// keeps an interrupted or failed write from replacing a previously saved set.
func saveDailyManifest(ctx context.Context, path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".turnsmith-daily-*")
	if err != nil {
		return fmt.Errorf("save practice set: %w", err)
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
	}()
	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("save practice set: %w", err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("save practice set: %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("save practice set: %w", err)
	}
	closed = true
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(temporary, path); err != nil {
		return fmt.Errorf("save practice set: %w", err)
	}
	return nil
}
