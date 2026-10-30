package cube333

import (
	"context"
	"fmt"

	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
)

type RandomMove struct{}

func (RandomMove) Descriptor() puzzle.GeneratorDescriptor {
	length := 25
	return puzzle.GeneratorDescriptor{ID: "random-move", AlgorithmVersion: "1", Description: "Filtered random outer-face moves", ExplicitLength: true, DefaultLength: &length, MinLength: 1, MaxLength: 1000, Guarantees: "Exact token count; no repeated adjacent face or A-opposite(A)-A window", Limitations: "Does not uniformly sample reachable states; not competition approved"}
}
func Registration() puzzle.Registration {
	return puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "333", Name: "3×3×3 cube", Description: "Standard Rubik's Cube", Aliases: []string{"3x3", "3x3x3"}, DefaultMethod: "random-move"}, Generators: []puzzle.Generator{RandomMove{}}}
}
func (RandomMove) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (puzzle.Generated, error) {
	if options.Length == nil || *options.Length < 1 || *options.Length > 1000 {
		return puzzle.Generated{}, fmt.Errorf("length must be between 1 and 1000")
	}
	moves := make([]cube.Move, 0, *options.Length)
	for i := 0; i < *options.Length; i++ {
		if err := ctx.Err(); err != nil {
			return puzzle.Generated{}, err
		}
		var candidates [6]cube.Face
		n := 0
		for face := cube.U; face <= cube.B; face++ {
			if i > 0 && moves[i-1].Face == face {
				continue
			}
			if i > 1 && moves[i-2].Face == face && moves[i-1].Face == face.Opposite() {
				continue
			}
			candidates[n] = face
			n++
		}
		face := candidates[random.Below(rng, uint64(n))]
		turn := cube.Turn(random.Below(rng, 3) + 1)
		moves = append(moves, cube.Move{Face: face, Layers: 1, Turns: turn})
	}
	tokens := make([]string, len(moves))
	for i, m := range moves {
		if err := ctx.Err(); err != nil {
			return puzzle.Generated{}, err
		}
		token, err := m.Notation(3)
		if err != nil {
			return puzzle.Generated{}, fmt.Errorf("render cube move: %w", err)
		}
		tokens[i] = token
	}
	return puzzle.Generated{Moves: tokens}, nil
}
