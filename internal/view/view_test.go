package view

import (
	"context"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cubestate"
	"strings"
	"testing"
)

func TestLayoutsAndNavigation(t *testing.T) {
	moves, _ := alg.Parse("R U2 F'", 100)
	for _, size := range []int{2, 3, 4} {
		states, e := cubestate.Replay(size, moves)
		if e != nil {
			t.Fatal(e)
		}
		width := 2*max(10, size*3) + 2
		height := 3*(size+1) + 6
		for _, action := range []string{"start", "next", "previous", "reset", "end"} {
			text := Frame(states, moves, 0, width, height, action, Options{})
			if strings.Contains(text, "Resize") {
				t.Fatal(size, text)
			}
			for _, line := range strings.Split(text, "\n") {
				if len(line) > width {
					t.Fatalf("%d width %d: %q", size, width, line)
				}
			}
		}
		back := Frame(states, moves, 1, 80, 40, "previous", Options{})
		if !strings.Contains(back, "Undo: U2") {
			t.Fatal(back)
		}
		reset := Frame(states, moves, 0, 80, 40, "reset", Options{})
		if !strings.Contains(reset, "solve cube") {
			t.Fatal(reset)
		}
	}
}
func TestFinalOnlyAndColor(t *testing.T) {
	for _, size := range []int{2, 3, 4} {
		for _, depth := range []string{"16", "256", "truecolor"} {
			var b strings.Builder
			e := (Guide{Size: size}).RenderView(context.Background(), &b, []string{"R", "U"}, true, depth, true)
			if e != nil || strings.Contains(b.String(), "Step ") || strings.Count(b.String(), "U / top") != 1 || !strings.Contains(b.String(), "\x1b[") {
				t.Fatal(e, b.String())
			}
		}
	}
}
