package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/practice"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/frivas/turnsmith/internal/terminal"
	"io"
	"strconv"
	"strings"
	"time"
)

type solveTimer struct {
	stats               practice.Stats
	store               *practice.Store
	session             string
	inspection          bool
	generate            func() (sequenceInput, error)
	input               sequenceInput
	phase               string
	start, inspectStart time.Time
	elapsed             time.Duration
	penalty             practice.Penalty
	savedID             string
}

func (t *solveTimer) next() error {
	input, e := t.generate()
	if e != nil {
		return e
	}
	t.input = input
	t.phase = "ready"
	t.elapsed = 0
	t.penalty = practice.None
	t.savedID = ""
	return nil
}
func inspectionPenalty(d time.Duration) practice.Penalty {
	if d > 17*time.Second {
		return practice.DNF
	}
	if d > 15*time.Second {
		return practice.PlusTwo
	}
	return practice.None
}
func (t *solveTimer) handle(ctx context.Context, event terminal.Event) (terminal.Frame, error) {
	sequence := wrapTimerText(alg.Format(t.input.Sequences[0]), max(event.Width, 1))
	requiredHeight := strings.Count(sequence, "\n") + 11
	small := event.Width < 48 || event.Height < requiredHeight
	press := (event.Key == terminal.KeySpace || event.Key == terminal.KeyEnter) && (!small || t.phase == "running")
	if press {
		switch t.phase {
		case "ready":
			if t.inspection {
				t.phase = "inspection"
				t.inspectStart = event.At
			} else {
				t.phase = "running"
				t.start = event.At
			}
		case "inspection":
			t.penalty = inspectionPenalty(event.At.Sub(t.inspectStart))
			t.phase = "running"
			t.start = event.At
		case "running":
			t.elapsed = event.At.Sub(t.start)
			if t.elapsed <= 0 {
				return terminal.Frame{}, fmt.Errorf("timer duration must be positive")
			}
			s, e := t.store.Record(ctx, t.session, practice.Solve{CreatedAt: event.At, DurationNS: int64(t.elapsed), Penalty: t.penalty, Scramble: alg.Format(t.input.Sequences[0]), Puzzle: t.input.Puzzle, Method: t.input.Method, Seed: t.input.Seed, AlgorithmVersion: t.input.AlgorithmVersion})
			if e != nil {
				return terminal.Frame{}, e
			}
			t.savedID = s.ID
			t.phase = "saved"
			if session, e := t.store.Get(ctx, t.session); e == nil {
				t.stats = practice.Summarize(session.Solves)
			} else {
				return terminal.Frame{}, e
			}
		}
	}
	if event.Key == terminal.KeyNext && (t.phase == "saved" || t.phase == "ready") {
		if e := t.next(); e != nil {
			return terminal.Frame{}, e
		}
	}
	if event.Key == terminal.KeyRune && t.phase == "saved" {
		p := t.penalty
		switch event.Rune {
		case '2':
			p = practice.PlusTwo
		case 'd', 'D':
			p = practice.DNF
		case '0':
			p = practice.None
		}
		if p != t.penalty {
			if _, e := t.store.Update(ctx, t.session, t.savedID, practice.Edit{Penalty: &p}); e != nil {
				return terminal.Frame{}, e
			}
			t.penalty = p
			if session, e := t.store.Get(ctx, t.session); e == nil {
				t.stats = practice.Summarize(session.Solves)
			} else {
				return terminal.Frame{}, e
			}
		}
	}
	elapsed := t.elapsed
	status := t.phase
	if t.phase == "running" {
		elapsed = event.At.Sub(t.start)
	}
	if t.phase == "inspection" {
		elapsed = event.At.Sub(t.inspectStart)
		status = fmt.Sprintf("inspection (%s)", inspectionPenalty(elapsed))
	}
	text := fmt.Sprintf("Turnsmith - session %s\n%s (%s)\n%s\n%s  %s  penalty %s\nSolves %d  best %s\nao5 %s  ao12 %s\nSpace/Enter start-stop; n next\n2 +2; d DNF; 0 clear; q exit\nQuitting a running attempt discards it.\n", t.session, t.input.Puzzle, t.input.Method, sequence, status, elapsed.Truncate(time.Millisecond), t.penalty, t.stats.Count, metricText(t.stats.Best), metricText(t.stats.Ao5), metricText(t.stats.Ao12))
	if small {
		text = fmt.Sprintf("Resize to at least 48 x %d to start.\n%s %s\nSpace/Enter still stops a running solve.\nq exits; active attempt is discarded.\n", requiredHeight, status, elapsed.Truncate(time.Millisecond))
	}
	return terminal.Frame{Text: wrapTimerText(text, max(event.Width, 1))}, nil
}
func (c CLI) timerCommand(ctx context.Context, args []string, out io.Writer) error {
	o, e := options(args, map[string]string{"session": "default", "data-dir": "", "puzzle": "333", "method": "", "seed": "", "length": "", "inspection": "0"})
	if e != nil {
		return e
	}
	if o.yes("help") {
		return commandHelp(out, "timer")
	}
	if len(o.pos) > 0 {
		return app.Invalid("timer takes no positional arguments")
	}
	if o.values["inspection"] != "0" && o.values["inspection"] != "15" {
		return app.Invalid("--inspection must be 0 or 15")
	}
	if e = c.checkTerminal(out); e != nil {
		return e
	}
	// Validate/generate before creating optional local storage.
	var root uint64
	var attempt uint64
	generate := func() (sequenceInput, error) {
		if attempt > 0 {
			o.seen["seed"] = true
			o.values["seed"] = strconv.FormatUint(random.Derive(root, o.values["puzzle"], o.values["method"], "timer/v1", attempt).Uint64(), 10)
		}
		input, err := c.generatedSequence(ctx, o)
		if err == nil {
			if attempt == 0 {
				root, _ = random.ParseSeed(input.Seed)
			}
			attempt++
		}
		return input, err
	}
	input, e := generate()
	if e != nil {
		return e
	}
	store, e := practice.New(o.values["data-dir"])
	if e != nil {
		return e
	}
	if _, e = store.Get(ctx, o.values["session"]); errors.Is(e, practice.ErrNotFound) {
		_, e = store.Create(ctx, o.values["session"])
	}
	if e != nil {
		return e
	}
	session, e := store.Get(ctx, o.values["session"])
	if e != nil {
		return e
	}
	t := solveTimer{stats: practice.Summarize(session.Solves), store: store, session: o.values["session"], inspection: o.values["inspection"] == "15", generate: generate, input: input, phase: "ready", penalty: practice.None}
	err := terminal.Run(ctx, c.Input, out, terminal.Options{Tick: 50 * time.Millisecond}, func(ev terminal.Event) (terminal.Frame, error) { return t.handle(ctx, ev) })
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Session %s: %d saved solves.\n", t.session, t.stats.Count)
	return err
}

// wrapTimerText keeps complete tokens visible; words longer than the terminal
// width are split so even user-defined session names cannot hide controls.
func wrapTimerText(text string, width int) string {
	width = max(width, 1)
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		current := ""
		for _, word := range strings.Fields(line) {
			for len([]rune(word)) > width {
				if current != "" {
					lines = append(lines, current)
					current = ""
				}
				r := []rune(word)
				lines = append(lines, string(r[:width]))
				word = string(r[width:])
			}
			if word == "" {
				continue
			}
			if current != "" && len([]rune(current))+1+len([]rune(word)) > width {
				lines = append(lines, current)
				current = ""
			}
			if current != "" {
				current += " "
			}
			current += word
		}
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
