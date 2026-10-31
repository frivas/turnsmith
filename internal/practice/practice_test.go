package practice

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func storeFor(t *testing.T) *Store {
	t.Helper()
	store, err := New(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func readJSON(t *testing.T, store *Store) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := store.ExportJSON(context.Background(), &data); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func sampleSolve() Solve {
	return Solve{DurationNS: 12345678901, Scramble: "R U R' U'", Puzzle: "333", Method: "random-move", AlgorithmVersion: "1", Seed: "18446744073709551615"}
}

func TestNoImplicitIOAndCRUD(t *testing.T) {
	ctx := context.Background()
	s := storeFor(t)
	if _, err := os.Stat(s.Dir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("New performed IO", err)
	}
	created, err := s.Create(ctx, "daily practice")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, "daily practice"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	stored, err := s.Record(ctx, created.ID, sampleSolve())
	if err != nil || stored.ID == "" || stored.CreatedAt.IsZero() || stored.Penalty != None {
		t.Fatal(stored, err)
	}
	penalty, notes := PlusTwo, "first line,\nsecond \"line\""
	edited, err := s.Update(ctx, created.Name, stored.ID, Edit{Penalty: &penalty, Notes: &notes})
	if err != nil || edited.Notes != notes || edited.Penalty != PlusTwo {
		t.Fatal(edited, err)
	}
	session, err := s.Get(ctx, created.Name)
	if err != nil || len(session.Solves) != 1 {
		t.Fatal(session, err)
	}
	session.Solves[0].Notes = "not saved"
	session, err = s.Get(ctx, created.Name)
	if err != nil || session.Solves[0].Notes != notes {
		t.Fatal(session, err)
	}
	before := readJSON(t, s)
	bad := Penalty("bad")
	if _, err := s.Update(ctx, created.Name, stored.ID, Edit{Penalty: &bad}); err == nil {
		t.Fatal("invalid edit accepted")
	}
	if !bytes.Equal(before, readJSON(t, s)) {
		t.Fatal("failed edit modified data")
	}
	if err := s.Delete(ctx, created.Name, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, created.ID, stored.ID); err != nil {
		t.Fatal(err)
	}
	session, err = s.Get(ctx, created.Name)
	if err != nil || len(session.Solves) != 0 {
		t.Fatal(session, err)
	}
	if err := s.DeleteSession(ctx, created.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, created.Name); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestAveragesPenaltiesAndDNF(t *testing.T) {
	makeSolves := func(n int) []Solve {
		result := make([]Solve, n)
		for i := range result {
			result[i] = Solve{DurationNS: int64(i+1) * int64(time.Second), Penalty: None}
		}
		return result
	}
	if stats := Summarize(nil); stats.Count != 0 || stats.Ao5.Available || stats.Best.Available {
		t.Fatal(stats)
	}
	solves := makeSolves(5)
	stats := Summarize(solves)
	if stats.Ao5.DurationNS != int64(3*time.Second) || stats.Ao12.Available || stats.Best.DurationNS != int64(time.Second) || stats.Worst.DurationNS != int64(5*time.Second) {
		t.Fatal(stats)
	}
	solves[0].Penalty = PlusTwo
	if got := Summarize(solves).Ao5.DurationNS; got != 3333333333 {
		t.Fatal(got)
	}
	solves[0].Penalty = DNF
	stats = Summarize(solves)
	if stats.Ao5.DNF || stats.Ao5.DurationNS != int64(4*time.Second) || stats.DNFCount != 1 || stats.Best.DurationNS != int64(2*time.Second) || stats.Worst.DurationNS != int64(5*time.Second) {
		t.Fatal(stats)
	}
	solves[1].Penalty = DNF
	if !Summarize(solves).Ao5.DNF {
		t.Fatal("two DNFs must invalidate ao5")
	}
	for i := range solves {
		solves[i].Penalty = DNF
	}
	stats = Summarize(solves)
	if !stats.Best.DNF || !stats.Worst.DNF || stats.DNFCount != 5 {
		t.Fatal(stats)
	}
	twelve := makeSolves(12)
	if got := Summarize(twelve).Ao12.DurationNS; got != 6500000000 {
		t.Fatal(got)
	}
	twelve[11].Penalty = DNF
	if stats := Summarize(twelve); stats.Ao12.DNF || stats.Ao12.DurationNS != 6500000000 {
		t.Fatal(stats)
	}
	twelve[10].Penalty = DNF
	if !Summarize(twelve).Ao12.DNF {
		t.Fatal("two DNFs must invalidate ao12")
	}
	thirteen := makeSolves(13)
	if got := Summarize(thirteen).Ao12.DurationNS; got != 7500000000 {
		t.Fatal("not a trailing window", got)
	}
	large := makeSolves(5)
	for i := range large {
		large[i].DurationNS = math.MaxInt64 - int64(2*time.Second) - 1
	}
	if got := Summarize(large).Ao5.DurationNS; got != large[0].DurationNS {
		t.Fatal("overflow in average", got)
	}
}

func TestJSONAndCSVRoundTripAndMerge(t *testing.T) {
	ctx := context.Background()
	s := storeFor(t)
	if _, err := s.Create(ctx, "empty session"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, "precision, session"); err != nil {
		t.Fatal(err)
	}
	solve := sampleSolve()
	solve.DurationNS = 9007199254740993
	solve.CreatedAt = time.Date(2026, 10, 31, 1, 2, 3, 987654321, time.FixedZone("offset", 20700))
	solve.Notes = "comma, quote \" and\nnew line"
	if _, err := s.Record(ctx, "precision, session", solve); err != nil {
		t.Fatal(err)
	}
	jsonData := readJSON(t, s)
	if !bytes.Contains(jsonData, []byte(`"duration_ns": "9007199254740993"`)) {
		t.Fatal(string(jsonData))
	}
	for _, format := range []string{"json", "csv"} {
		t.Run(format, func(t *testing.T) {
			var exported bytes.Buffer
			other := storeFor(t)
			var summary ImportSummary
			var err error
			if format == "json" {
				err = s.ExportJSON(ctx, &exported)
				if err == nil {
					summary, err = other.ImportJSON(ctx, bytes.NewReader(exported.Bytes()), Merge)
				}
			} else {
				err = s.ExportCSV(ctx, &exported)
				if err == nil {
					summary, err = other.ImportCSV(ctx, bytes.NewReader(exported.Bytes()), Merge)
				}
			}
			if err != nil || summary.SessionsAdded != 2 || summary.SolvesAdded != 1 {
				t.Fatal(summary, err)
			}
			if !bytes.Equal(jsonData, readJSON(t, other)) {
				t.Fatal("roundtrip changed data", string(readJSON(t, other)))
			}
			summary, err = other.ImportJSON(ctx, bytes.NewReader(jsonData), Merge)
			if err != nil || summary.Duplicates != 1 || summary.SolvesAdded != 0 || summary.SessionsAdded != 0 {
				t.Fatal(summary, err)
			}
			var doc Document
			if err := json.Unmarshal(jsonData, &doc); err != nil {
				t.Fatal(err)
			}
			doc.Sessions[1].Solves[0].Notes = "conflicting change"
			conflict, _ := json.Marshal(doc)
			if _, err := other.ImportJSON(ctx, bytes.NewReader(conflict), Merge); !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			if !bytes.Equal(jsonData, readJSON(t, other)) {
				t.Fatal("failed merge modified store")
			}
			if _, err := other.ImportJSON(ctx, bytes.NewReader(conflict), Replace); err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(jsonData, readJSON(t, other)) {
				t.Fatal("explicit replacement did not happen")
			}
			if err := other.Recover(ctx); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(jsonData, readJSON(t, other)) {
				t.Fatal("backup did not restore prior state")
			}
		})
	}
}

func TestMalformedImportsDoNotChangeStore(t *testing.T) {
	ctx := context.Background()
	s := storeFor(t)
	if _, err := s.Create(ctx, "keep"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Record(ctx, "keep", sampleSolve()); err != nil {
		t.Fatal(err)
	}
	before := readJSON(t, s)
	for _, data := range []string{
		`{}`, `{"schema_version":2,"sessions":[]}`, `{"schema_version":1,"sessions":null}`,
		`{"schema_version":1,"schema_version":1,"sessions":[]}`, `{"schema_version":1,"sessions":[],"unknown":true}`,
		`{"schema_version":1,"sessions":[]} {"schema_version":1,"sessions":[]}`,
		strings.Replace(string(before), `"12345678901"`, `"12.5"`, 1),
		strings.Replace(string(before), `"none"`, `"DNS"`, 1),
		strings.Replace(string(before), `"12345678901"`, `12345678901`, 1),
	} {
		if _, err := s.ImportJSON(ctx, strings.NewReader(data), Replace); err == nil {
			t.Fatal("accepted malformed import", data)
		}
		if !bytes.Equal(before, readJSON(t, s)) {
			t.Fatal("malformed import changed existing data")
		}
	}
	var csvData bytes.Buffer
	if err := s.ExportCSV(ctx, &csvData); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(bytes.NewReader(csvData.Bytes())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		row, column int
		value       string
	}{{0, 0, "bad"}, {1, 0, "2"}, {2, 7, "1.5"}, {2, 8, "DNS"}, {2, 2, "missing"}, {2, 1, "unknown"}} {
		copyRows := make([][]string, len(records))
		for i := range records {
			copyRows[i] = append([]string{}, records[i]...)
		}
		copyRows[tc.row][tc.column] = tc.value
		var invalid bytes.Buffer
		writer := csv.NewWriter(&invalid)
		_ = writer.WriteAll(copyRows)
		if _, err := s.ImportCSV(ctx, &invalid, Replace); err == nil {
			t.Fatal("accepted invalid CSV", tc)
		}
		if !bytes.Equal(before, readJSON(t, s)) {
			t.Fatal("invalid CSV changed existing data")
		}
	}
}

func TestCorruptionFailedWritesAndRecovery(t *testing.T) {
	ctx := context.Background()
	s := storeFor(t)
	if _, err := s.Create(ctx, "safe"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Record(ctx, "safe", sampleSolve()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.BackupPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(s.BackupPath(), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Record(ctx, "safe", sampleSolve()); err == nil {
		t.Fatal("backup failure ignored")
	}
	after, _ := os.ReadFile(s.Path())
	if !bytes.Equal(before, after) {
		t.Fatal("failed backup replaced main data")
	}
	if err := os.Remove(s.BackupPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Record(ctx, "safe", sampleSolve()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.Path(), []byte("broken data"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sessions(ctx); !errors.Is(err, ErrCorrupt) {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, "must not reset"); !errors.Is(err, ErrCorrupt) {
		t.Fatal(err)
	}
	if err := s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if restored, err := s.Get(ctx, "safe"); err != nil || len(restored.Solves) != 1 {
		t.Fatal(restored, err)
	}
	if err := os.Remove(s.Path()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sessions(ctx); !errors.Is(err, ErrCorrupt) {
		t.Fatal("missing main silently reset despite backup", err)
	}
	if err := s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(s.Dir(), ".practice-write-*"))
	if len(matches) != 0 {
		t.Fatal("temporary writes leaked", matches)
	}
}

func TestLockCancellationAndOwnership(t *testing.T) {
	s := storeFor(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Sessions(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Dir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled call created data dir")
	}
	token, err := s.acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	other, _ := New(s.Dir())
	other.lockWait = 80 * time.Millisecond
	if _, err := other.Sessions(context.Background()); !errors.Is(err, ErrLocked) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := other.Sessions(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	owner, err := s.LockInfo()
	if err != nil || owner.Token != token {
		t.Fatal(owner, err)
	}
	if err := s.BreakLock("wrong-owner"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := s.release(token); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Sessions(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	s := storeFor(t)
	if _, err := s.Create(ctx, "parallel"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errorsCh := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			other, err := New(s.Dir())
			if err != nil {
				errorsCh <- err
				return
			}
			for j := 0; j < 5; j++ {
				if _, err := other.Record(ctx, "parallel", sampleSolve()); err != nil {
					errorsCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
	session, err := s.Get(ctx, "parallel")
	if err != nil || len(session.Solves) != 30 {
		t.Fatal("lost concurrent updates", len(session.Solves), err)
	}
}

func TestConcurrentProcesses(t *testing.T) {
	s := storeFor(t)
	if _, err := s.Create(context.Background(), "processes"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errorsCh := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			command := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$")
			command.Env = append(os.Environ(), "TURNSMITH_PRACTICE_WORKER=1", "TURNSMITH_PRACTICE_DIR="+s.Dir())
			if data, err := command.CombinedOutput(); err != nil {
				errorsCh <- fmt.Errorf("child: %w: %s", err, data)
			}
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
	session, err := s.Get(context.Background(), "processes")
	if err != nil || len(session.Solves) != 12 {
		t.Fatal("lost cross-process updates", len(session.Solves), err)
	}
}

func TestProcessHelper(t *testing.T) {
	if os.Getenv("TURNSMITH_PRACTICE_WORKER") != "1" {
		t.Skip("subprocess helper")
	}
	s, err := New(os.Getenv("TURNSMITH_PRACTICE_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.Record(context.Background(), "processes", sampleSolve()); err != nil {
			t.Fatal(err)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestExportErrors(t *testing.T) {
	s := storeFor(t)
	if _, err := s.Create(context.Background(), "export"); err != nil {
		t.Fatal(err)
	}
	if err := s.ExportJSON(context.Background(), failingWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if err := s.ExportCSV(context.Background(), failingWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

type mustNotRead struct{}

func (mustNotRead) Read([]byte) (int, error) { panic("canceled import read input") }

func TestCanceledImportDoesNotRead(t *testing.T) {
	s := storeFor(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ImportJSON(ctx, mustNotRead{}, Merge); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.ImportCSV(ctx, mustNotRead{}, Merge); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Dir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled import created files", err)
	}
}
