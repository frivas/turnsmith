package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/daily"
	"github.com/frivas/turnsmith/internal/puzzle"
)

func dailyCLI(t *testing.T) CLI {
	t.Helper()
	registry, err := puzzle.New(cube333.Registration())
	if err != nil {
		t.Fatal(err)
	}
	return CLI{Service: app.Service{Registry: registry, Version: "test"}}
}

func TestDailyCommandReproducibleManifest(t *testing.T) {
	c := dailyCLI(t)
	args := []string{"--date", "2026-10-31", "--name", "Weekend practice", "--puzzle", "3x3", "--count", "2", "--length", "5", "--json"}
	var first, second bytes.Buffer
	if err := c.dailyCommand(context.Background(), args, &first); err != nil {
		t.Fatal(err)
	}
	if err := c.dailyCommand(context.Background(), args, &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("same daily inputs changed")
	}
	var manifest daily.Manifest
	if err := json.Unmarshal(first.Bytes(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Kind != "practice-set" || manifest.DatePolicy != "UTC" || manifest.Name != "Weekend practice" || manifest.Result.Puzzle != "333" || len(manifest.Result.Scrambles) != 2 {
		t.Fatalf("%+v", manifest)
	}
	if len(manifest.Result.Scrambles[0].Moves) != 5 {
		t.Fatal(manifest.Result.Scrambles)
	}
	var renamed bytes.Buffer
	if err := c.dailyCommand(context.Background(), []string{"--date", "2026-10-31", "--name", "Different name", "--count", "1", "--length", "5", "--json"}, &renamed); err != nil {
		t.Fatal(err)
	}
	var other daily.Manifest
	_ = json.Unmarshal(renamed.Bytes(), &other)
	if other.Result.Seed != manifest.Result.Seed || !reflect.DeepEqual(other.Result.Scrambles[0], manifest.Result.Scrambles[0]) {
		t.Fatal("name/count changed the daily seed or prefix")
	}
}

func TestDailyValidationDoesNotTouchOutput(t *testing.T) {
	c := dailyCLI(t)
	directory := t.TempDir()
	path := filepath.Join(directory, "daily.json")
	if err := os.WriteFile(path, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, invalidArgs := range [][]string{{"--date", "2026-02-30"}, {"--count", "0"}, {"--count", "10001"}, {"--length", "1000", "--count", "1001"}, {"--name", "bad\x1bname"}, {"--name", strings.Repeat("x", 257)}, {"--puzzle", ""}, {"--method", ""}, {"--seed", "42"}, {"--output", ""}, {"unexpected"}} {
		var out bytes.Buffer
		args := append([]string{"--output", path}, invalidArgs...)
		err := c.dailyCommand(context.Background(), args, &out)
		var invalid *app.InvalidInput
		if !errors.As(err, &invalid) || out.Len() != 0 {
			t.Fatalf("%v: %v %q", args, err, out.String())
		}
		content, _ := os.ReadFile(path)
		if string(content) != "previous" {
			t.Fatal("saved output changed on invalid arguments")
		}
	}
	var oversized bytes.Buffer
	if err := c.dailyCommand(context.Background(), []string{"--output", path, "--count", "1000", "--length", "100"}, &oversized); err == nil || !strings.Contains(err.Error(), "1 MiB") || oversized.Len() != 0 {
		t.Fatal(err, oversized.Len())
	}
	content, _ := os.ReadFile(path)
	if string(content) != "previous" {
		t.Fatal("oversized set replaced the saved output")
	}
	files, _ := os.ReadDir(directory)
	if len(files) != 1 {
		t.Fatal("validation left temporary files", files)
	}
}

func TestDailySavesJSONAndPrintsReadableMetadata(t *testing.T) {
	c := dailyCLI(t)
	path := filepath.Join(t.TempDir(), "set.json")
	var out bytes.Buffer
	if err := os.WriteFile(path, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.dailyCommand(context.Background(), []string{"--date", "2026-10-31", "--count", "1", "--output", path}, &out); err != nil {
		t.Fatal(err)
	}
	var manifest daily.Manifest
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"2026-10-31 (UTC)", "seed: " + manifest.Result.Seed, "method: random-move", "algorithm:", "1. " + manifest.Result.Scrambles[0].Sequence} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing %q: %q", text, out.String())
		}
	}
	var failed bytes.Buffer
	if err := c.dailyCommand(context.Background(), []string{"--date", "2026-10-31", "--output", filepath.Join(path, "invalid")}, &failed); err == nil || failed.Len() != 0 {
		t.Fatal(err, failed.String())
	}
}

func TestDailyCanceledDoesNotSave(t *testing.T) {
	c := dailyCLI(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(t.TempDir(), "set.json")
	var out bytes.Buffer
	if err := c.dailyCommand(ctx, []string{"--output", path}, &out); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
}
