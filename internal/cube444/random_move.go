// Package cube444 generates native 4x4 outer and two-layer wide moves.
package cube444

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
)

type RandomMove struct{}

func (RandomMove) Descriptor() puzzle.GeneratorDescriptor {
	length := 40
	return puzzle.GeneratorDescriptor{ID: "random-move", AlgorithmVersion: "1", Description: "Filtered random outer and two-layer wide moves", ExplicitLength: true, DefaultLength: &length, MinLength: 1, MaxLength: 1000, Guarantees: "Exact token count; one- and two-layer turns; no adjacent repeated face or A-opposite(A)-A window", Limitations: "Does not uniformly sample reachable states; not competition approved"}
}
func Registration() puzzle.Registration {
	return puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "444", Name: "4×4×4 cube", Description: "Four-layer cube with outer and wide turns", Aliases: []string{"4x4", "4x4x4"}, DefaultMethod: "random-move"}, Generators: []puzzle.Generator{RandomMove{}}}
}
func draw(ctx context.Context, rng random.Source, n uint64) (uint64, error) {
	threshold := -n % n
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		v := rng.Uint64()
		if v >= threshold {
			return v % n, nil
		}
	}
}
func (RandomMove) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (puzzle.Generated, error) {
	if options.Length == nil || *options.Length < 1 || *options.Length > 1000 {
		return puzzle.Generated{}, fmt.Errorf("length must be between 1 and 1000")
	}
	tokens := make([]string, 0, *options.Length)
	faces := make([]cube.Face, 0, *options.Length)
	for i := 0; i < *options.Length; i++ {
		var candidates [6]cube.Face
		n := 0
		for face := cube.U; face <= cube.B; face++ {
			if i > 0 && faces[i-1] == face {
				continue
			}
			if i > 1 && faces[i-2] == face && faces[i-1] == face.Opposite() {
				continue
			}
			candidates[n] = face
			n++
		}
		choice, err := draw(ctx, rng, uint64(n))
		if err != nil {
			return puzzle.Generated{}, err
		}
		face := candidates[choice]
		width, err := draw(ctx, rng, 2)
		if err != nil {
			return puzzle.Generated{}, err
		}
		turn, err := draw(ctx, rng, 3)
		if err != nil {
			return puzzle.Generated{}, err
		}
		token, err := (cube.Move{Face: face, Layers: uint8(width + 1), Turns: cube.Turn(turn + 1)}).Notation(4)
		if err != nil {
			return puzzle.Generated{}, err
		}
		faces = append(faces, face)
		tokens = append(tokens, token)
	}
	return puzzle.Generated{Moves: tokens}, nil
}
