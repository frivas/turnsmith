package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/frivas/turnsmith/internal/app"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func savedSequence(t *testing.T, counts ...int) string {
	t.Helper()
	batch := app.Result{SchemaVersion: 1, Puzzle: "333", Method: "random-move", AlgorithmVersion: "1", Seed: "0", Parameters: app.Parameters{Count: len(counts)}}
	for i, count := range counts {
		moves := make([]string, count)
		for j := range moves {
			moves[j] = "R"
			if j%2 != 0 {
				moves[j] = "U"
			}
		}
		batch.Scrambles = append(batch.Scrambles, app.Scramble{Index: i + 1, Moves: moves, MoveCount: count, Sequence: strings.Join(moves, " ")})
	}
	data, _ := json.Marshal(batch)
	path := filepath.Join(t.TempDir(), "saved.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestInverseJSONPreservesLongSavedSequence(t *testing.T) {
	file := savedSequence(t, 1001)
	var out bytes.Buffer
	if err := (CLI{}).sequenceCommand(context.Background(), "inverse", []string{"--file", file, "--json"}, &out); err != nil {
		t.Fatal(err)
	}
	var got inspectionDocument
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Sequences) != 1 || got.Sequences[0].MoveCount != 1001 || got.Sequences[0].Moves[0] != "R'" || got.Sequences[0].Solved {
		t.Fatal("long inverse truncated or changed", got.Sequences)
	}
}
func TestInspectStepBoundsBeforeOutput(t *testing.T) {
	for _, counts := range [][]int{{1001}, {501, 501}} {
		file := savedSequence(t, counts...)
		var out bytes.Buffer
		err := (CLI{}).sequenceCommand(context.Background(), "inspect", []string{"--file", file, "--steps"}, &out)
		if err == nil || out.Len() != 0 {
			t.Fatalf("step overflow wrote partial output: %v %q", err, out.String())
		}
	}
}
func TestSavedEmptySequenceAndUnusedFlags(t *testing.T) {
	file := savedSequence(t, 0)
	var out bytes.Buffer
	if err := (CLI{}).sequenceCommand(context.Background(), "inspect", []string{"--file", file}, &out); err == nil {
		t.Fatal("accepted empty saved scramble")
	}
	for _, args := range [][]string{{"--alg", "R", "--steps"}, {"--alg", "R", "--diagram"}, {"--alg", "R", "--color", "always"}, {"--alg", "R", "--color-depth", "16"}} {
		if err := (CLI{}).sequenceCommand(context.Background(), "inverse", args, &out); err == nil {
			t.Fatal("inverse accepted unused flag", args)
		}
	}
	if err := (CLI{}).sequenceCommand(context.Background(), "guide", []string{"--index", "1"}, &out); err == nil || !strings.Contains(err.Error(), "requires saved") {
		t.Fatal("guide silently ignored index", err)
	}
}
