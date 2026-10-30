package cube333

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/frivas/turnsmith/internal/cube"
)

// StepGuide renders state independently of generation and its random stream.
type StepGuide struct{}

func (StepGuide) Render(ctx context.Context, w io.Writer, tokens []string, color bool) error {
	moves := make([]cube.Move, len(tokens))
	for i, token := range tokens {
		move, err := parseMove(token)
		if err != nil {
			return err
		}
		moves[i] = move
	}
	state := Solved()
	for i := 0; i <= len(moves); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		var text strings.Builder
		if i == 0 {
			text.WriteString("Start solved. Hold the white center on top (U), green center facing you (F).\n")
			text.WriteString("Keep this orientation; turn the named layer, not the whole cube.\n")
			text.WriteString("Clockwise is viewed looking directly at the face being turned.\n")
			text.WriteString("Colors: W=white, Y=yellow, O=orange, R=red, G=green, B=blue.\n")
			text.WriteString("Diagrams assume the standard color scheme; labels name fixed face positions.\n\n")
			fmt.Fprintf(&text, "Step 0/%d: solved\n", len(moves))
		} else {
			if err := state.Apply(moves[i-1]); err != nil {
				return err
			}
			face := [6]string{"top", "bottom", "left", "right", "front", "back"}[moves[i-1].Face]
			turn := [3]string{"90 degrees clockwise", "180 degrees", "90 degrees counterclockwise"}[moves[i-1].Turns-1]
			fmt.Fprintf(&text, "Step %d/%d: %s - turn the %s face %s\n", i, len(moves), tokens[i-1], face, turn)
		}
		state.net(&text, color)
		text.WriteByte('\n')
		if _, err := io.WriteString(w, text.String()); err != nil {
			return err
		}
	}
	return nil
}

func (s State) net(w *strings.Builder, color bool) {
	row := func(face cube.Face, r int) {
		// Colors and readable letters stay paired even without ANSI support.
		palette := [6]string{"48;2;245;245;245;30", "48;2;255;219;77;30", "48;2;247;145;50;30", "48;2;201;54;62;97", "48;2;32;132;90;97", "48;2;40;98;168;97"}
		for c := 0; c < 3; c++ {
			value := s[face][r*3+c]
			if color {
				fmt.Fprintf(w, "\x1b[%sm[%c]\x1b[0m", palette[value], "WYORGB"[value])
			} else {
				fmt.Fprintf(w, "[%c]", "WYORGB"[value])
			}
		}
	}
	w.WriteString("           U / top\n")
	for r := 0; r < 3; r++ {
		w.WriteString("           ")
		row(cube.U, r)
		w.WriteByte('\n')
	}
	w.WriteString("L / left   F / front  R / right  B / back\n")
	for r := 0; r < 3; r++ {
		for i, face := range []cube.Face{cube.L, cube.F, cube.R, cube.B} {
			if i > 0 {
				w.WriteString("  ")
			}
			row(face, r)
		}
		w.WriteByte('\n')
	}
	w.WriteString("           D / bottom\n")
	for r := 0; r < 3; r++ {
		w.WriteString("           ")
		row(cube.D, r)
		w.WriteByte('\n')
	}
}
