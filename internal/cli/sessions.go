package cli

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/practice"
	"io"
	"os"
	"strings"
	"time"
)

func secondsNS(text string) (int64, error) {
	if text == "" {
		return 0, app.Invalid("--time requires decimal seconds, for example 12.345")
	}
	parts := strings.Split(text, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, app.Invalid("--time requires decimal seconds")
	}
	for _, r := range text {
		if (r < '0' || r > '9') && r != '.' {
			return 0, app.Invalid("--time requires nonnegative decimal seconds")
		}
	}
	if len(parts) == 2 && (parts[1] == "" || len(parts[1]) > 9) {
		return 0, app.Invalid("--time supports at most 9 fractional digits")
	}
	d, e := time.ParseDuration(text + "s")
	if e != nil {
		return 0, app.Invalid("invalid --time: %v", e)
	}
	return int64(d), nil
}
func parsePenalty(text string) (practice.Penalty, error) {
	switch text {
	case "none":
		return practice.None, nil
	case "+2":
		return practice.PlusTwo, nil
	case "DNF":
		return practice.DNF, nil
	}
	return "", app.Invalid("--penalty must be none, +2, or DNF")
}
func metricText(m practice.Metric) string {
	if !m.Available {
		return "—"
	}
	if m.DNF {
		return "DNF"
	}
	return time.Duration(m.DurationNS).String()
}
func (c CLI) sessionsCommand(ctx context.Context, args []string, out io.Writer) error {
	o, e := options(args, map[string]string{"data-dir": "", "session": "", "id": "", "time": "", "scramble": "", "penalty": "none", "notes": "", "json": "false", "format": "json", "file": "", "replace": "false", "token": "", "puzzle": "", "method": "", "seed": "", "algorithm-version": ""}, "json", "replace")
	if e != nil {
		return e
	}
	if o.yes("help") {
		return commandHelp(out, "sessions")
	}
	action := "list"
	if len(o.pos) > 0 {
		action = o.pos[0]
		o.pos = o.pos[1:]
	}
	if len(o.pos) > 1 {
		return app.Invalid("expected one session name")
	}
	name := o.values["session"]
	if len(o.pos) == 1 {
		if o.seen["session"] {
			return app.Invalid("supply the session once")
		}
		name = o.pos[0]
	}
	switch action {
	case "list", "new", "show", "add", "edit", "delete", "delete-session", "export", "import", "recover", "lock", "unlock":
	default:
		return app.Invalid("unknown sessions action %q", action)
	}
	if name == "" {
		name = "default"
	}
	if o.values["format"] != "json" && o.values["format"] != "csv" {
		return app.Invalid("--format must be json or csv")
	}
	// Reject flags that could otherwise appear to succeed while silently doing nothing.
	allowed := map[string]bool{"data-dir": true, "help": true, "json": true, "session": true}
	for _, k := range map[string][]string{"add": {"time", "scramble", "penalty", "notes", "puzzle", "method", "seed", "algorithm-version"}, "edit": {"id", "time", "penalty", "notes"}, "delete": {"id"}, "export": {"format", "file"}, "import": {"format", "file", "replace"}, "unlock": {"token"}}[action] {
		allowed[k] = true
	}
	for k := range o.seen {
		if !allowed[k] {
			return app.Invalid("--%s is not available for sessions %s", k, action)
		}
	}
	if (action == "delete" || action == "edit") && o.values["id"] == "" {
		return app.Invalid("--id is required")
	}
	var duration int64
	var penalty practice.Penalty
	if o.seen["time"] {
		duration, e = secondsNS(o.values["time"])
		if e != nil {
			return e
		}
	}
	if action == "add" || o.seen["penalty"] {
		penalty, e = parsePenalty(o.values["penalty"])
		if e != nil {
			return e
		}
	}
	if action == "add" && (!o.seen["time"] || !o.seen["scramble"]) {
		return app.Invalid("add requires --time and --scramble")
	}
	if action == "edit" && !o.seen["time"] && !o.seen["penalty"] && !o.seen["notes"] {
		return app.Invalid("edit requires --time, --penalty, or --notes")
	}
	store, e := practice.New(o.values["data-dir"])
	if e != nil {
		return e
	}
	var value any
	switch action {
	case "list":
		value, e = store.Sessions(ctx)
	case "new":
		value, e = store.Create(ctx, name)
	case "show":
		var s practice.Session
		s, e = store.Get(ctx, name)
		value = struct {
			Session practice.Session `json:"session"`
			Stats   practice.Stats   `json:"stats"`
		}{s, practice.Summarize(s.Solves)}
		if e == nil && !o.yes("json") {
			st := practice.Summarize(s.Solves)
			if _, e = fmt.Fprintf(out, "%s (%s): %d solves, %d DNF\nBest: %s  Worst: %s  ao5: %s  ao12: %s\n", s.Name, s.ID, st.Count, st.DNFCount, metricText(st.Best), metricText(st.Worst), metricText(st.Ao5), metricText(st.Ao12)); e != nil {
				return e
			}
			for _, v := range s.Solves {
				if _, e = fmt.Fprintf(out, "%s  %s  %s  %s  %s\n", v.ID, v.CreatedAt.Format(time.RFC3339Nano), time.Duration(v.DurationNS), v.Penalty, v.Notes); e != nil {
					return e
				}
			}
			return nil
		}
	case "add":
		value, e = store.Record(ctx, name, practice.Solve{DurationNS: duration, Penalty: penalty, Scramble: o.values["scramble"], Notes: o.values["notes"], Puzzle: o.values["puzzle"], Method: o.values["method"], Seed: o.values["seed"], AlgorithmVersion: o.values["algorithm-version"]})
	case "edit":
		ed := practice.Edit{}
		if o.seen["time"] {
			ed.DurationNS = &duration
		}
		if o.seen["penalty"] {
			ed.Penalty = &penalty
		}
		if o.seen["notes"] {
			n := o.values["notes"]
			ed.Notes = &n
		}
		value, e = store.Update(ctx, name, o.values["id"], ed)
	case "delete":
		e = store.Delete(ctx, name, o.values["id"])
	case "delete-session":
		e = store.DeleteSession(ctx, name)
	case "recover":
		e = store.Recover(ctx)
	case "lock":
		value, e = store.LockInfo()
	case "unlock":
		e = store.BreakLock(o.values["token"])
	case "export":
		w := out
		if o.seen["file"] {
			f, err := os.OpenFile(o.values["file"], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
			defer func() {
				if e != nil {
					_ = os.Remove(o.values["file"])
				}
			}()
		}
		if o.values["format"] == "csv" {
			e = store.ExportCSV(ctx, w)
		} else {
			e = store.ExportJSON(ctx, w)
		}
		return e
	case "import":
		r := c.Input
		if o.seen["file"] {
			f, err := os.Open(o.values["file"])
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		}
		if r == nil {
			return app.Invalid("import requires --file or stdin")
		}
		mode := practice.Merge
		if o.yes("replace") {
			mode = practice.Replace
		}
		if o.values["format"] == "csv" {
			value, e = store.ImportCSV(ctx, r, mode)
		} else {
			value, e = store.ImportJSON(ctx, r, mode)
		}
	}
	if e != nil {
		return e
	}
	if value == nil {
		value = map[string]bool{"ok": true}
	}
	return writeJSON(out, value)
}
