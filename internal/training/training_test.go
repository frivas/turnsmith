package training

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func collection() Collection {
	return Collection{SchemaVersion: 1, Name: "My cases", Cases: []Case{{ID: "a", Name: "One", Puzzle: "333", Algorithm: "R U R'"}, {ID: "b", Name: "Two", Puzzle: "444", Algorithm: "x Rw U2 Rw' x'"}, {ID: "c", Name: "Three", Puzzle: "222", Algorithm: "[R,U]"}}}
}
func store(t *testing.T) *Store {
	t.Helper()
	s, e := NewStore(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func source() random.Source { return random.Derive(7, "training", "selection", "1", 0) }
func TestCollectionAndSetup(t *testing.T) {
	c := collection()
	data, _ := json.Marshal(c)
	got, e := LoadCollection(bytes.NewReader(data))
	if e != nil || !reflect.DeepEqual(got, c) {
		t.Fatal(e)
	}
	for _, item := range c.Cases {
		setup, e := item.Setup()
		if e != nil {
			t.Fatal(e)
		}
		moves, e := alg.Parse(item.Algorithm, 10000)
		if e != nil {
			t.Fatal(e)
		}
		size, _ := item.Size()
		states, e := cubestate.Replay(size, append(setup, moves...))
		if e != nil || !states[len(states)-1].Solved() {
			t.Fatal("setup did not solve", item, e)
		}
	}
	for _, text := range []string{`{"schema_version":1,"schema_version":1,"name":"x","cases":[]}`, `{"schema_version":2,"name":"x","cases":[]}`, string(data) + "{}", strings.Replace(string(data), `"algorithm":"R U R'"`, `"algorithm":"bad"`, 1)} {
		if _, e := LoadCollection(strings.NewReader(text)); e == nil {
			t.Fatal("accepted bad collection", text)
		}
	}
	duplicate := c
	duplicate.Cases = append(duplicate.Cases, duplicate.Cases[0])
	if e := duplicate.Validate(); e == nil {
		t.Fatal("duplicate case accepted")
	}
}
func attempt(id string, n int64) Attempt {
	return Attempt{ID: id, At: time.Date(2026, 10, 31, 0, 0, int(n), 0, time.UTC), DurationNS: n * int64(time.Second), Success: true}
}
func TestProgressAndSelection(t *testing.T) {
	s := store(t)
	c := collection()
	ctx := context.Background()
	p, e := s.Progress(ctx, c)
	if e != nil || len(p) != 3 || p[0].Status != New {
		t.Fatal(p, e)
	}
	first, e := s.Next(ctx, c, Sequential, source())
	if e != nil || first.ID != "a" {
		t.Fatal(first, e)
	}
	if e := s.Record(ctx, c, "a", attempt("1", 1)); e != nil {
		t.Fatal(e)
	}
	if e := s.Record(ctx, c, "a", attempt("1", 1)); e != nil {
		t.Fatal("not idempotent", e)
	}
	next, e := s.Next(ctx, c, Sequential, source())
	if e != nil || next.ID != "b" {
		t.Fatal(next, e)
	}
	balanced, e := s.Next(ctx, c, Balanced, source())
	if e != nil || balanced.ID == "a" {
		t.Fatal(balanced, e)
	}
	if e := s.SetStatus(ctx, c, "b", Known); e != nil {
		t.Fatal(e)
	}
	if e := s.SetStatus(ctx, c, "a", Known); e != nil {
		t.Fatal(e)
	}
	learning, e := s.Next(ctx, c, LearningFirst, source())
	if e != nil || learning.ID != "c" {
		t.Fatal(learning, e)
	}
	if e := s.Record(ctx, c, "b", attempt("2", 20)); e != nil {
		t.Fatal(e)
	}
	if e := s.Record(ctx, c, "c", attempt("3", 5)); e != nil {
		t.Fatal(e)
	}
	slow, e := s.Next(ctx, c, Slowest, source())
	if e != nil || slow.ID != "b" {
		t.Fatal(slow, e)
	}
	p, e = s.Progress(ctx, c)
	if e != nil || len(p[0].Attempts) != 1 {
		t.Fatal(p, e)
	}
	changed := collection()
	changed.Cases[0].Algorithm = "R2"
	fresh, e := s.Progress(ctx, changed)
	if e != nil || len(fresh[0].Attempts) != 0 {
		t.Fatal("edited collection reused history", e)
	}
	for i := 0; i < 10; i++ {
		a, _ := s.Next(ctx, c, Balanced, source())
		b, _ := s.Next(ctx, c, Balanced, source())
		if a.ID != b.ID {
			t.Fatal("non-deterministic selection")
		}
	}
}
func TestConcurrentAndExchange(t *testing.T) {
	s := store(t)
	c := collection()
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			other, e := NewStore(s.dir)
			if e == nil {
				e = other.Record(ctx, c, "a", attempt(fmt.Sprint(i), int64(i)))
			}
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	p, e := s.Progress(ctx, c)
	if e != nil || len(p[0].Attempts) != 20 {
		t.Fatal("lost update", len(p[0].Attempts), e)
	}
	var export bytes.Buffer
	if e := s.ExportJSON(ctx, &export); e != nil {
		t.Fatal(e)
	}
	copy := store(t)
	if e := copy.ImportJSON(ctx, bytes.NewReader(export.Bytes()), false); e != nil {
		t.Fatal(e)
	}
	if e := copy.ImportJSON(ctx, bytes.NewReader(export.Bytes()), false); e != nil {
		t.Fatal(e)
	}
	q, e := copy.Progress(ctx, c)
	if e != nil || !reflect.DeepEqual(p, q) {
		t.Fatal("round trip", e)
	}
	if e := copy.Record(ctx, c, "a", attempt("extra", 30)); e != nil {
		t.Fatal(e)
	}
	if e := copy.ImportJSON(ctx, bytes.NewReader(export.Bytes()), false); !errors.Is(e, ErrConflict) {
		t.Fatal("did not reject import conflict", e)
	}
	if e := copy.ImportJSON(ctx, bytes.NewReader(export.Bytes()), true); e != nil {
		t.Fatal(e)
	}
}
func TestCorruptionAndRecovery(t *testing.T) {
	s := store(t)
	c := collection()
	ctx := context.Background()
	if e := s.Record(ctx, c, "a", attempt("1", 1)); e != nil {
		t.Fatal(e)
	}
	if e := s.Record(ctx, c, "a", attempt("2", 2)); e != nil {
		t.Fatal(e)
	}
	broken := []byte("{broken")
	if e := os.WriteFile(s.Path(), broken, 0600); e != nil {
		t.Fatal(e)
	}
	if e := s.Record(ctx, c, "a", attempt("3", 3)); !errors.Is(e, ErrCorrupt) {
		t.Fatal("corruption reset", e)
	}
	data, _ := os.ReadFile(s.Path())
	if !bytes.Equal(data, broken) {
		t.Fatal("corrupt bytes changed")
	}
	if e := s.Recover(ctx); e != nil {
		t.Fatal(e)
	}
	p, e := s.Progress(ctx, c)
	if e != nil || len(p[0].Attempts) != 1 {
		t.Fatal("backup not restored", e)
	}
	matches, _ := os.ReadDir(s.dir)
	preserved := false
	for _, entry := range matches {
		if strings.Contains(entry.Name(), "before-recovery") {
			preserved = true
		}
	}
	if !preserved {
		t.Fatal("corrupt primary not preserved")
	}
	if e := os.Remove(s.Path()); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Progress(ctx, c); !errors.Is(e, ErrCorrupt) {
		t.Fatal("missing primary reset", e)
	}
}
func TestLockAndCancel(t *testing.T) {
	s := store(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Progress(ctx, collection()); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if e := os.MkdirAll(s.lockPath(), 0700); e != nil {
		t.Fatal(e)
	}
	owner := LockOwner{Token: "owner", PID: 123, CreatedAt: time.Now()}
	data, _ := json.Marshal(owner)
	if e := os.WriteFile(s.lockPath()+"/owner.json", data, 0600); e != nil {
		t.Fatal(e)
	}
	short, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, e := s.Progress(short, collection()); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	if e := s.BreakLock("wrong"); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if e := s.BreakLock("owner"); e != nil {
		t.Fatal(e)
	}
}
