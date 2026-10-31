package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestCompletionShellsAndValues(t *testing.T) {
	c := dailyCLI(t)
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			var out bytes.Buffer
			if err := c.completionCommand([]string{shell}, &out); err != nil {
				t.Fatal(err)
			}
			for _, word := range []string{"turnsmith", "daily", "sessions", "inspect", "completion", "random-move", "333", "3x3", "color-depth", "truecolor"} {
				if !strings.Contains(out.String(), word) {
					t.Errorf("missing %q", word)
				}
			}
			if shell == "bash" || shell == "zsh" || shell == "fish" {
				if executable, err := exec.LookPath(shell); err == nil {
					command := exec.Command(executable, "-n")
					command.Stdin = strings.NewReader(out.String())
					if result, err := command.CombinedOutput(); err != nil {
						t.Fatalf("invalid shell syntax: %v\n%s", err, result)
					}
				}
			}
		})
	}
	for _, args := range [][]string{nil, {"nu"}, {"bash", "extra"}} {
		var out bytes.Buffer
		if err := c.completionCommand(args, &out); err == nil || out.Len() != 0 {
			t.Fatal(args, err, out.String())
		}
	}
}

func TestBashCompletionBehavior(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	c := dailyCLI(t)
	var script bytes.Buffer
	if err := c.completionCommand([]string{"bash"}, &script); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		words  string
		index  string
		want   string
		absent string
	}{
		{"turnsmith da", "1", "daily", "--method"},
		{"turnsmith daily --puzzle 3", "3", "333", "random-move"},
		{"turnsmith daily --method r", "3", "random-move", "333"},
		{"turnsmith daily --o", "2", "--output", "--steps"},
		{"turnsmith completion f", "2", "fish", "--output"},
	} {
		command := exec.Command(bash, "-c", script.String()+"\nCOMP_WORDS=("+tc.words+"); COMP_CWORD="+tc.index+"; _turnsmith; printf '%s\\n' \"${COMPREPLY[@]}\"")
		result, err := command.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(result))
		}
		if !strings.Contains(string(result), tc.want) || strings.Contains(string(result), tc.absent) {
			t.Fatalf("%s: %q", tc.words, result)
		}
	}
}
