package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cli"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
)

func runner(t testing.TB) cli.CLI {
	t.Helper()
	r, err := puzzle.New(cube333.Registration())
	if err != nil {
		t.Fatal(err)
	}
	return cli.CLI{Service: app.Service{Registry: r, Entropy: bytes.NewReader(make([]byte, 8)), Version: "0.1.0"}, Build: cli.Build{Version: "0.1.0", Commit: "test", Date: "unknown"}}
}
func run(t testing.TB, args ...string) (int, string, string) {
	t.Helper()
	c := runner(t)
	var out, err bytes.Buffer
	status := c.Run(context.Background(), args, &out, &err)
	return status, out.String(), err.String()
}
func TestDefault(t *testing.T) {
	status, out, stderr := run(t)
	if status != 0 || stderr != "" || len(strings.Fields(out)) != 25 || !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 1 {
		t.Fatal(status, out, stderr)
	}
}
func TestInvalidArguments(t *testing.T) {
	cases := [][]string{
		{"--length", "0"}, {"--length", "-1"}, {"--length", "1001"}, {"--length", "1.2"}, {"--length="}, {"--length", "+1"}, {"--length", "18446744073709551616"},
		{"--count", "0"}, {"--count", "-1"}, {"--count", "10001"}, {"--count", "1.5"}, {"--count", "wat"}, {"--count="},
		{"--length", "1000", "--count", "1001"},
		{"--seed", "-1"}, {"--seed", "+1"}, {"--seed", "0x42"}, {"--seed", "1.2"}, {"--seed="}, {"--seed", "18446744073709551616"}, {"--seed", " 42"},
		{"--puzzle", "444"}, {"--puzzle", "pyraminx"}, {"--puzzle="}, {"--method", "random-state"}, {"--method="},
		{"generate"}, {"foo"}, {"--seed", "42", "extra"}, {"--unknown"}, {"--count"}, {"--length", "--json"}, {"--"}, {"--json=wat"}, {"--version", "--json"},
		{"--count", "2", "-n", "3"}, {"puzzles", "--seed", "42"}, {"puzzles", "--count", "2"}, {"version", "extra"}, {"help", "extra"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			status, out, stderr := run(t, args...)
			if status != 2 || out != "" || !strings.HasPrefix(stderr, "error: ") || strings.Contains(stderr, "panic") {
				t.Fatal(status, out, stderr)
			}
		})
	}
}
func TestErrorMessages(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--puzzle", "444"}, "error: unsupported puzzle \"444\"; run \"turnsmith puzzles\" to list supported puzzles\n"},
		{[]string{"--method", "random-state"}, "error: puzzle \"333\" does not support method \"random-state\"\n"},
		{[]string{"--length", "0"}, "error: --length must be an integer between 1 and 1000\n"},
	} {
		_, _, got := run(t, tc.args...)
		if got != tc.want {
			t.Fatalf("%v: %q", tc.args, got)
		}
	}
}
func TestHelpVersionDiscovery(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"puzzles", "--help"}, {"version"}, {"--version"}, {"puzzles"}, {"puzzles", "--json"}} {
		status, out, stderr := run(t, args...)
		if status != 0 || out == "" || stderr != "" {
			t.Fatal(args, status, out, stderr)
		}
	}
	_, out, _ := run(t, "--version")
	if out != "turnsmith 0.1.0\n" {
		t.Fatal(out)
	}
	_, out, _ = run(t, "version")
	if out != "turnsmith 0.1.0\ncommit: test\nbuilt: unknown\n" {
		t.Fatal(out)
	}
	_, out, _ = run(t, "puzzles", "--json")
	var listing struct {
		SchemaVersion int                 `json:"schema_version"`
		Puzzles       []puzzle.Descriptor `json:"puzzles"`
	}
	if err := json.Unmarshal([]byte(out), &listing); err != nil {
		t.Fatal(err)
	}
	if listing.SchemaVersion != 1 || len(listing.Puzzles) != 1 || listing.Puzzles[0].ID != "333" || !listing.Puzzles[0].Methods[0].ExplicitLength {
		t.Fatal(out)
	}
}
func TestJSONAndText(t *testing.T) {
	for _, seed := range []string{"0", "42", "18446744073709551615"} {
		status, text, stderr := run(t, "--seed", seed, "--count", "3", "--length", "30")
		if status != 0 || stderr != "" {
			t.Fatal(status, stderr)
		}
		status, encoded, stderr := run(t, "--seed="+seed, "-n=3", "-l=30", "--json", "-p", "3x3x3")
		if status != 0 || stderr != "" {
			t.Fatal(status, stderr)
		}
		var result app.Result
		if err := json.Unmarshal([]byte(encoded), &result); err != nil {
			t.Fatal(err)
		}
		if result.Seed != seed || result.SchemaVersion != 1 || result.Puzzle != "333" || result.AppVersion != "0.1.0" || result.Method != "random-move" || result.AlgorithmVersion != "1" || result.Parameters.Count != 3 || *result.Parameters.Length != 30 || len(result.Scrambles) != 3 || !strings.HasSuffix(encoded, "\n") {
			t.Fatal(encoded)
		}
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		for i, s := range result.Scrambles {
			if s.Index != i+1 || s.Sequence != lines[i] || s.MoveCount != 30 || len(s.Moves) != 30 || strings.Join(s.Moves, " ") != s.Sequence {
				t.Fatal(encoded, text)
			}
		}
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestFailures(t *testing.T) {
	c := runner(t)
	c.Service.Entropy = strings.NewReader("")
	var out, stderr bytes.Buffer
	if status := c.Run(context.Background(), nil, &out, &stderr); status != 1 || out.Len() != 0 || !strings.Contains(stderr.String(), "acquire root seed") {
		t.Fatal(status, out.String(), stderr.String())
	}
	// Every rendering path must surface writer errors.
	for _, args := range [][]string{{"--seed", "0"}, {"--seed", "0", "--json"}, {"puzzles"}, {"puzzles", "--json"}, {"help"}, {"version"}, {"--version"}} {
		stderr.Reset()
		if status := c.Run(context.Background(), args, failWriter{}, &stderr); status != 1 || !strings.Contains(stderr.String(), "write output") {
			t.Fatal(args, status, stderr.String())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out.Reset()
	stderr.Reset()
	if status := c.Run(ctx, nil, &out, &stderr); status != 130 || out.Len() != 0 {
		t.Fatal(status, out.String(), stderr.String())
	}
}
func TestGoldenVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		Seed      string   `json:"seed"`
		Length    int      `json:"length"`
		Sequences []string `json:"sequences"`
	}
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		args := []string{"--seed", v.Seed, "--length", fmt.Sprint(v.Length), "--count", fmt.Sprint(len(v.Sequences))}
		status, out, stderr := run(t, args...)
		if status != 0 || stderr != "" || out != strings.Join(v.Sequences, "\n")+"\n" {
			t.Fatalf("vector %s/%d mismatch:\n%s\n%s", v.Seed, v.Length, out, stderr)
		}
	}
}

type secondCallFailure struct{ calls *int }

func (secondCallFailure) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "test", AlgorithmVersion: "1"}
}
func (g secondCallFailure) Generate(context.Context, random.Source, puzzle.Options) (puzzle.Generated, error) {
	*g.calls++
	if *g.calls == 2 {
		return puzzle.Generated{}, errors.New("fixture failure")
	}
	return puzzle.Generated{Moves: []string{"tip++"}}, nil
}
func TestNoPartialBatchOnFailure(t *testing.T) {
	for _, jsonFlag := range []bool{false, true} {
		calls := 0
		r, err := puzzle.New(puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "test", Name: "test", DefaultMethod: "test"}, Generators: []puzzle.Generator{secondCallFailure{&calls}}})
		if err != nil {
			t.Fatal(err)
		}
		c := runner(t)
		c.Service.Registry = r
		var out, stderr bytes.Buffer
		args := []string{"-p", "test", "-n", "3", "--seed", "0"}
		if jsonFlag {
			args = append(args, "--json")
		}
		status := c.Run(context.Background(), args, &out, &stderr)
		if status != 1 || out.Len() != 0 || !strings.Contains(stderr.String(), "generate scramble 2") {
			t.Fatal(status, out.String(), stderr.String())
		}
	}
}
func TestForeignPuzzleUsesUnchangedCLIAndRenderers(t *testing.T) {
	calls := 0
	r, err := puzzle.New(puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "test", Name: "test", DefaultMethod: "test"}, Generators: []puzzle.Generator{secondCallFailure{&calls}}})
	if err != nil {
		t.Fatal(err)
	}
	c := runner(t)
	c.Service.Registry = r
	var out, stderr bytes.Buffer
	status := c.Run(context.Background(), []string{"-p", "test", "--seed", "0", "--json"}, &out, &stderr)
	if status != 0 {
		t.Fatal(status, stderr.String())
	}
	var got app.Result
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Parameters.Length != nil || !reflect.DeepEqual(got.Scrambles[0].Moves, []string{"tip++"}) {
		t.Fatal(got)
	}
	out.Reset()
	stderr.Reset()
	status = c.Run(context.Background(), []string{"-p", "test", "-l", "25"}, &out, &stderr)
	if status != 2 || out.Len() != 0 || !strings.Contains(stderr.String(), "does not support --length") {
		t.Fatal(status, stderr.String())
	}
}
