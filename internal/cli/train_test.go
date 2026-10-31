package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/frivas/turnsmith/internal/terminal"
	"github.com/frivas/turnsmith/internal/training"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func trainCollection() training.Collection {
	return training.Collection{SchemaVersion: 1, Name: "User cases", Cases: []training.Case{{ID: "a", Name: "Example", Puzzle: "333", Algorithm: "x R U R' x'"}, {ID: "b", Name: "Wide", Puzzle: "444", Algorithm: "Rw U Rw'"}}}
}
func trainFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cases.json")
	data, _ := json.Marshal(trainCollection())
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	return path, filepath.Join(dir, "data")
}
func TestTrainCommands(t *testing.T) {
	file, dir := trainFixture(t)
	c := CLI{}
	ctx := context.Background()
	var out bytes.Buffer
	if e := c.trainCommand(ctx, []string{"list", "--file", file, "--data-dir", dir, "--json"}, &out); e != nil || !strings.Contains(out.String(), `"status": "new"`) {
		t.Fatal(out.String(), e)
	}
	out.Reset()
	if e := c.trainCommand(ctx, []string{"status", "--file", file, "--data-dir", dir, "--case", "a", "--status", "known"}, &out); e != nil {
		t.Fatal(e)
	}
	progress := filepath.Join(t.TempDir(), "progress.json")
	if e := c.trainCommand(ctx, []string{"export", "--data-dir", dir, "--output", progress}, &out); e != nil {
		t.Fatal(e)
	}
	imported := t.TempDir()
	if e := c.trainCommand(ctx, []string{"import", "--data-dir", imported, "--input", progress}, &out); e != nil {
		t.Fatal(e)
	}
	out.Reset()
	if e := c.trainCommand(ctx, []string{"--list", "--file", file, "--data-dir", imported}, &out); e != nil || !strings.Contains(out.String(), "known") {
		t.Fatal(out.String(), e)
	}
	before, _ := os.ReadFile(file)
	if e := c.trainCommand(ctx, []string{"export", "--data-dir", dir, "--file", file, "--output", filepath.Join(filepath.Dir(file), ".", "cases.json")}, &out); e == nil {
		t.Fatal("overwrote collection")
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("collection changed")
	}
}
func TestTrainInvalidAndTTYGate(t *testing.T) {
	file, dir := trainFixture(t)
	for _, args := range [][]string{{"--file", file, "--data-dir", dir}, {"list"}, {"wat"}, {"list", "extra"}, {"list", "--file", file, "--mode", "slowest"}, {"status", "--file", file, "--case", "a"}, {"--list", "--export-progress"}, {"import"}, {"unlock"}} {
		var out bytes.Buffer
		if e := (CLI{}).trainCommand(context.Background(), args, &out); e == nil {
			t.Fatal("accepted", args)
		}
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("TTY rejection created training directory", e)
	}
}
func newTrainModel(t *testing.T) *trainModel {
	t.Helper()
	s, e := training.NewStore(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	m := &trainModel{ctx: context.Background(), store: s, collection: trainCollection(), policy: training.Sequential, rng: random.Derive(1, "training", "selection", "1", 0)}
	if e := m.next(); e != nil {
		t.Fatal(e)
	}
	return m
}
func trainEvent(key terminal.Key, r rune, at time.Time) terminal.Event {
	return terminal.Event{Key: key, Rune: r, At: at, Width: 80, Height: 40}
}
func TestTrainTimedHintedAttempts(t *testing.T) {
	m := newTrainModel(t)
	start := time.Now()
	if _, e := m.handle(trainEvent(terminal.KeySpace, 0, start)); e != nil {
		t.Fatal(e)
	}
	_, _ = m.handle(trainEvent(terminal.KeyRune, 'h', start.Add(time.Second)))
	_, _ = m.handle(trainEvent(terminal.KeySpace, 0, start.Add(2*time.Second)))
	if !m.pending {
		t.Fatal("not waiting for correctness")
	}
	p, _ := m.store.Progress(m.ctx, m.collection)
	if len(p[0].Attempts) != 0 {
		t.Fatal("recorded before correctness")
	}
	_, _ = m.handle(trainEvent(terminal.KeyRune, 'y', start.Add(3*time.Second)))
	p, _ = m.store.Progress(m.ctx, m.collection)
	if len(p[0].Attempts) != 1 || p[0].Attempts[0].DurationNS != int64(2*time.Second) || !p[0].Attempts[0].Hint || !p[0].Attempts[0].Success {
		t.Fatal(p)
	}
	_, _ = m.handle(trainEvent(terminal.KeySpace, 0, start.Add(4*time.Second)))
	_, _ = m.handle(trainEvent(terminal.KeyRune, 'd', start.Add(5*time.Second)))
	p, _ = m.store.Progress(m.ctx, m.collection)
	if len(p[0].Attempts) != 2 || p[0].Attempts[1].Success || p[0].Attempts[1].Hint {
		t.Fatal(p)
	}
	_, _ = m.handle(trainEvent(terminal.KeyRune, 'k', start))
	p, _ = m.store.Progress(m.ctx, m.collection)
	if p[0].Status != training.Known {
		t.Fatal("status not saved")
	}
	_, _ = m.handle(trainEvent(terminal.KeyNext, 'n', start))
	if m.item.ID != "b" {
		t.Fatal("next case", m.item)
	}
}
func TestTrainSmallScreenAndPaging(t *testing.T) {
	m := newTrainModel(t)
	start := time.Now()
	small := trainEvent(terminal.KeySpace, 0, start)
	small.Width = 20
	small.Height = 10
	frame, e := m.handle(small)
	if e != nil || !m.started.IsZero() || !strings.Contains(frame.Text, "Resize") {
		t.Fatal(frame, e)
	}
	_, _ = m.handle(trainEvent(terminal.KeySpace, 0, start))
	small.At = start.Add(time.Second)
	_, _ = m.handle(small)
	if !m.pending {
		t.Fatal("small screen could not stop")
	}
	small.Key = terminal.KeyRune
	small.Rune = 'd'
	_, _ = m.handle(small)
	p, _ := m.store.Progress(m.ctx, m.collection)
	if len(p[0].Attempts) != 1 || p[0].Attempts[0].Success {
		t.Fatal("small screen could not record", p)
	}
	m.item.Algorithm = strings.Repeat("R U ", 100)
	m.setup, _ = m.item.Setup()
	frame, _ = m.handle(trainEvent(terminal.KeyInitial, 0, start))
	if !strings.Contains(frame.Text, "Setup page 1/") {
		t.Fatal(frame.Text)
	}
	_, _ = m.handle(trainEvent(terminal.KeyNext, 0, start))
	if m.setupPage != 1 {
		t.Fatal("setup did not page")
	}
	for _, line := range trainPages([]string{"R", "U2", "Rw'"}, 5) {
		if len(line) > 5 {
			t.Fatal("bad wrap", line)
		}
	}
}
