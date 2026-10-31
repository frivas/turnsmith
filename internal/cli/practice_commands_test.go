package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/practice"
	"github.com/frivas/turnsmith/internal/terminal"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionsCommandsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "data")
	c := CLI{}
	var out bytes.Buffer
	run := func(args ...string) {
		t.Helper()
		out.Reset()
		args = append(args, "--data-dir", dir)
		if e := c.sessionsCommand(ctx, args, &out); e != nil {
			t.Fatal(args, e)
		}
	}
	run("new", "work")
	run("add", "work", "--time", "12.345678901", "--scramble", "R U", "--notes", "hello", "--json")
	var solve practice.Solve
	if e := json.Unmarshal(out.Bytes(), &solve); e != nil {
		t.Fatal(e)
	}
	if solve.DurationNS != 12345678901 {
		t.Fatal(solve)
	}
	run("edit", "work", "--id", solve.ID, "--penalty", "+2", "--notes", "")
	run("show", "work", "--json")
	if !strings.Contains(out.String(), `"penalty": "+2"`) {
		t.Fatal(out.String())
	}
	run("export", "--format", "csv")
	csv := out.String()
	other := filepath.Join(t.TempDir(), "other")
	c.Input = strings.NewReader(csv)
	out.Reset()
	if e := c.sessionsCommand(ctx, []string{"import", "--data-dir", other, "--format", "csv"}, &out); e != nil {
		t.Fatal(e)
	}
	store, _ := practice.New(other)
	session, e := store.Get(ctx, "work")
	if e != nil || len(session.Solves) != 1 || session.Solves[0].Notes != "" {
		t.Fatal(session, e)
	}
	run("delete", "work", "--id", solve.ID)
	run("delete-session", "work")
}
func TestPracticeValidationAndNoPipeStorage(t *testing.T) {
	for _, value := range []string{"-1", "1e2", "1.0000000001", "1.", ".2", "1h"} {
		if _, e := secondsNS(value); e == nil {
			t.Fatal(value)
		}
	}
	dir := filepath.Join(t.TempDir(), "absent")
	c := CLI{Input: strings.NewReader("")}
	var out bytes.Buffer
	if e := c.timerCommand(context.Background(), []string{"--data-dir", dir}, &out); e == nil {
		t.Fatal("piped timer accepted")
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("timer wrote storage", e)
	}
	for _, args := range [][]string{{"add", "--time", "1"}, {"edit", "--id", "a"}, {"list", "--replace"}, {"import", "--format", "wat"}, {"unlock"}} {
		out.Reset()
		args = append(args, "--data-dir", dir)
		if e := c.sessionsCommand(context.Background(), args, &out); e == nil {
			t.Fatal(args)
		}
	}
}
func TestTimerLifecycleAndInspection(t *testing.T) {
	ctx := context.Background()
	s, _ := practice.New(t.TempDir())
	_, _ = s.Create(ctx, "work")
	moves, _ := alg.Parse("R U", 10)
	tm := solveTimer{store: s, session: "work", inspection: true, phase: "ready", input: sequenceInput{Puzzle: "333", Method: "random-move", Seed: "42", Sequences: [][]alg.Move{moves}}, generate: func() (sequenceInput, error) {
		return sequenceInput{Puzzle: "333", Sequences: [][]alg.Move{moves}}, nil
	}}
	now := time.Now()
	event := func(key terminal.Key, r rune, d time.Duration) {
		t.Helper()
		if _, e := tm.handle(ctx, terminal.Event{Key: key, Rune: r, At: now.Add(d), Width: 80, Height: 24}); e != nil {
			t.Fatal(e)
		}
	}
	event(terminal.KeySpace, 0, 0)
	event(terminal.KeyEnter, 0, 16*time.Second)
	event(terminal.KeySpace, 0, 28*time.Second)
	event(terminal.KeyTick, 0, 29*time.Second)
	event(terminal.KeyEnter, 0, 30*time.Second)
	session, _ := s.Get(ctx, "work")
	if len(session.Solves) != 1 || session.Solves[0].DurationNS != int64(12*time.Second) || session.Solves[0].Penalty != practice.PlusTwo || session.Solves[0].Seed != "42" {
		t.Fatal(session)
	}
	event(terminal.KeyRune, 'd', 31*time.Second)
	session, _ = s.Get(ctx, "work")
	if session.Solves[0].Penalty != practice.DNF {
		t.Fatal(session)
	}
	event(terminal.KeyNext, 0, 32*time.Second)
	if tm.phase != "ready" || tm.savedID != "" {
		t.Fatal(tm)
	}
	for _, tc := range []struct {
		d time.Duration
		p practice.Penalty
	}{{15 * time.Second, practice.None}, {15*time.Second + 1, practice.PlusTwo}, {17 * time.Second, practice.PlusTwo}, {17*time.Second + 1, practice.DNF}} {
		if inspectionPenalty(tc.d) != tc.p {
			t.Fatal(tc)
		}
	}
}

func TestTimerWrapAndResize(t *testing.T) {
	ctx := context.Background()
	store, _ := practice.New(t.TempDir())
	_, _ = store.Create(ctx, "test")
	algorithm := strings.TrimSpace(strings.Repeat("Rw U2 F' Lw2 D B ", 7))
	moves, _ := alg.Parse(algorithm, 100)
	tm := solveTimer{store: store, session: "test", phase: "ready", penalty: practice.None, input: sequenceInput{Puzzle: "444", Sequences: [][]alg.Move{moves}}}
	now := time.Now()
	frame, e := tm.handle(ctx, terminal.Event{Key: terminal.KeyInitial, At: now, Width: 48, Height: 30})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(strings.Join(strings.Fields(frame.Text), " "), alg.Format(moves)) {
		t.Fatal("scramble was truncated", frame.Text)
	}
	for _, line := range strings.Split(frame.Text, "\n") {
		if len([]rune(line)) > 48 {
			t.Fatal("line overflow", line)
		}
	}
	_, e = tm.handle(ctx, terminal.Event{Key: terminal.KeySpace, At: now, Width: 20, Height: 6})
	if e != nil || tm.phase != "ready" {
		t.Fatal(tm.phase, e)
	}
	_, e = tm.handle(ctx, terminal.Event{Key: terminal.KeySpace, At: now, Width: 80, Height: 30})
	if e != nil || tm.phase != "running" {
		t.Fatal(tm.phase, e)
	}
	_, e = tm.handle(ctx, terminal.Event{Key: terminal.KeySpace, At: now.Add(time.Second), Width: 20, Height: 6})
	if e != nil || tm.phase != "saved" {
		t.Fatal(tm.phase, e)
	}
	s, _ := store.Get(ctx, "test")
	if len(s.Solves) != 1 {
		t.Fatal(s)
	}
}
