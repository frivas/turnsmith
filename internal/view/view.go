// Package view renders cube snapshots without influencing algorithm generation.
package view

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/cubestate"
	"io"
	"strings"
)

type Options struct {
	Color bool
	Depth string
	Width int
}

func row(b *strings.Builder, s cubestate.State, face cube.Face, r int, o Options) {
	palette := [6]string{"48;5;255;30", "48;5;221;30", "48;5;208;30", "48;5;160;97", "48;5;29;97", "48;5;25;97"}
	if o.Depth == "16" {
		palette = [6]string{"107;30", "103;30", "43;30", "41;97", "42;97", "44;97"}
	}
	if o.Depth == "truecolor" {
		palette = [6]string{"48;2;245;245;245;30", "48;2;255;219;77;30", "48;2;247;145;50;30", "48;2;201;54;62;97", "48;2;32;132;90;97", "48;2;40;98;168;97"}
	}
	for c := 0; c < s.Size; c++ {
		value := s.Faces[face][r*s.Size+c]
		if o.Color {
			fmt.Fprintf(b, "\x1b[%sm[%c]\x1b[0m", palette[value], "WYORGB"[value])
		} else {
			fmt.Fprintf(b, "[%c]", "WYORGB"[value])
		}
	}
}
func Net(s cubestate.State, o Options) string {
	var b strings.Builder
	width := max(10, s.Size*3)
	gap := 2
	if o.Width > 0 && o.Width < 4*width+3*gap {
		columns := 2
		if o.Width < 2*width+gap {
			columns = 1
		}
		faces := []cube.Face{cube.U, cube.D, cube.L, cube.R, cube.F, cube.B}
		names := [6]string{"U / top", "D / bottom", "L / left", "R / right", "F / front", "B / back"}
		for start := 0; start < len(faces); start += columns {
			end := min(start+columns, len(faces))
			for i := start; i < end; i++ {
				if i > start {
					b.WriteString("  ")
				}
				fmt.Fprintf(&b, "%-*s", width, names[faces[i]])
			}
			b.WriteByte('\n')
			for r := 0; r < s.Size; r++ {
				for i := start; i < end; i++ {
					if i > start {
						b.WriteString("  ")
					}
					row(&b, s, faces[i], r, o)
					b.WriteString(strings.Repeat(" ", width-s.Size*3))
				}
				b.WriteByte('\n')
			}
		}
		return b.String()
	}
	indent := strings.Repeat(" ", width+gap)
	fmt.Fprintf(&b, "%sU / top\n", indent)
	for r := 0; r < s.Size; r++ {
		b.WriteString(indent)
		row(&b, s, cube.U, r, o)
		b.WriteByte('\n')
	}
	for i, label := range []string{"L / left", "F / front", "R / right", "B / back"} {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%-*s", width, label)
	}
	b.WriteByte('\n')
	for r := 0; r < s.Size; r++ {
		for i, f := range []cube.Face{cube.L, cube.F, cube.R, cube.B} {
			if i > 0 {
				b.WriteString("  ")
			}
			row(&b, s, f, r, o)
			b.WriteString(strings.Repeat(" ", width-s.Size*3))
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "%sD / bottom\n", indent)
	for r := 0; r < s.Size; r++ {
		b.WriteString(indent)
		row(&b, s, cube.D, r, o)
		b.WriteByte('\n')
	}
	return b.String()
}
func Describe(m alg.Move) string {
	names := map[byte]string{'U': "top", 'D': "bottom", 'L': "left", 'R': "right", 'F': "front", 'B': "back", 'M': "middle (like L)", 'E': "equator (like D)", 'S': "standing (like F)", 'x': "whole cube (like R)", 'y': "whole cube (like U)", 'z': "whole cube (like F)"}
	turn := [3]string{"90 degrees clockwise", "180 degrees", "90 degrees counterclockwise"}[m.Turns-1]
	face := names[m.Face]
	if m.Width > 1 {
		face = fmt.Sprintf("%d-layer %s", m.Width, face)
	}
	return "turn the " + face + " face " + turn
}
func Orientation(size int) string {
	if size == 2 || size == 4 {
		return "Start solved. Hold the white face up (U), green face facing you (F).\n"
	}
	return "Start solved. Hold the white center on top (U), green center facing you (F).\n"
}

type Guide struct {
	Size  int
	Depth string
	Width int
}

func (g Guide) Render(ctx context.Context, w io.Writer, tokens []string, color bool) error {
	return g.RenderView(ctx, w, tokens, color, g.Depth, false)
}
func (g Guide) RenderView(ctx context.Context, w io.Writer, tokens []string, color bool, depth string, final bool) error {
	g.Depth = depth
	size := g.Size
	if size == 0 {
		size = 3
	}
	moves, err := alg.Parse(strings.Join(tokens, " "), 1000)
	if err != nil {
		return err
	}
	states, err := cubestate.Replay(size, moves)
	if err != nil {
		return err
	}
	if final {
		if e := ctx.Err(); e != nil {
			return e
		}
		_, e := io.WriteString(w, Orientation(size)+Net(states[len(states)-1], Options{Color: color, Depth: depth, Width: g.Width}))
		return e
	}
	for i, state := range states {
		if err := ctx.Err(); err != nil {
			return err
		}
		var b strings.Builder
		if i == 0 {
			b.WriteString(Orientation(size))
			b.WriteString("Keep this orientation; turn the named layer, not the whole cube (unless x/y/z).\nClockwise is viewed looking directly at the face being turned.\nColors: W=white, Y=yellow, O=orange, R=red, G=green, B=blue.\nDiagrams assume the standard color scheme; labels name fixed face positions.\n\n")
			fmt.Fprintf(&b, "Step 0/%d: solved\n", len(moves))
		} else {
			fmt.Fprintf(&b, "Step %d/%d: %s - %s\n", i, len(moves), moves[i-1].String(), Describe(moves[i-1]))
		}
		b.WriteString(Net(state, Options{Color: color, Depth: g.Depth, Width: g.Width}))
		b.WriteByte('\n')
		if _, err := io.WriteString(w, b.String()); err != nil {
			return err
		}
	}
	return nil
}
func Frame(states []cubestate.State, moves []alg.Move, index, width, height int, action string, o Options) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Guide %d/%d\nW up, G front at start\n", index, len(moves))
	instruction := "Start solved."
	if action == "previous" && index < len(moves) {
		instruction = "Undo: " + moves[index].Inverse().String()
	} else if action == "reset" {
		instruction = "Reset: solve cube"
	} else if action == "end" {
		instruction = "Final view; no turns"
	} else if index > 0 {
		instruction = "Apply: " + moves[index-1].String()
	}
	b.WriteString(instruction + "\n")
	if index < len(moves) {
		fmt.Fprintf(&b, "Next: [%s]\n", moves[index].String())
	} else {
		b.WriteString("Complete.\n")
	}
	o.Width = width
	b.WriteString(Net(states[index], o))
	b.WriteString("Right next | Left undo\nHome reset | q quit\n")
	if height > 0 && strings.Count(b.String(), "\n") > height {
		return fmt.Sprintf("Resize for full cube (%dx%d). q quits.\n", width, height)
	}
	return b.String()
}
