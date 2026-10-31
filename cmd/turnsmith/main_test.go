package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecutable(t *testing.T) {
	name := "turnsmith"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, ".")
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v\n%s", err, data)
	}
	for _, tc := range []struct {
		args   []string
		status int
	}{
		{nil, 0}, {[]string{"--seed", "42", "--json"}, 0}, {[]string{"--version"}, 0}, {[]string{"version"}, 0}, {[]string{"help"}, 0}, {[]string{"puzzles", "--json"}, 0},
		{[]string{"--seed", "-1"}, 2}, {[]string{"--puzzle", "555"}, 2},
		{[]string{"--steps", "--seed", "42", "--length", "3", "--color", "never"}, 0},
		{[]string{"--steps", "--json"}, 2},
		{[]string{"--puzzle", "222", "--seed", "42", "--json"}, 0},
		{[]string{"--puzzle", "444", "--seed", "42", "--diagram"}, 0},
		{[]string{"--method", "random-state", "--seed", "42", "--json"}, 0},
		{[]string{"inspect", "--alg", "[R,U]", "--json"}, 0},
		{[]string{"inverse", "--alg", "R U"}, 0},
		{[]string{"guide", "--seed", "42"}, 2},
	} {
		command := exec.Command(binary, tc.args...)
		var out, stderr bytes.Buffer
		command.Stdout = &out
		command.Stderr = &stderr
		err := command.Run()
		status := 0
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if status != tc.status {
			t.Fatal(tc.args, status, stderr.String())
		}
		if status != 0 {
			if out.Len() != 0 || !strings.HasPrefix(stderr.String(), "error: ") {
				t.Fatal(out.String(), stderr.String())
			}
			continue
		}
		if stderr.Len() != 0 {
			t.Fatal(stderr.String())
		}
		if len(tc.args) > 0 && tc.args[0] == "--steps" && !strings.Contains(out.String(), "Step 3/3: U") {
			t.Fatal(out.String())
		}
		if len(tc.args) == 0 && len(strings.Fields(out.String())) != 25 {
			t.Fatal(out.String())
		}
		if len(tc.args) > 0 && tc.args[len(tc.args)-1] == "--json" && !json.Valid(out.Bytes()) {
			t.Fatal(out.String())
		}
	}
}
