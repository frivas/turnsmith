package randomstate

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/unixpickle/gocube"
)

type Cube333 struct{}
type Cube222 struct{}

func (Cube333) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "random-state", AlgorithmVersion: "1", Description: "Uniform reachable 3×3 state, deterministic two-phase scramble", Guarantees: "Uniform reachable cubie-state sampling; sequence reaches the sampled state", Limitations: "Lazy solver tables and variable search cost; bounded search can fail; no competition approval"}
}
func (Cube222) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "random-state", AlgorithmVersion: "1", Description: "Uniform 2×2 state modulo whole-cube rotation", Guarantees: "Uniform sampling of 3674160 states with a fixed BLD corner; sequence reaches the sampled state", Limitations: "Lazy distance table; no competition approval"}
}
func Registration222() puzzle.Registration {
	return puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "222", Name: "2×2×2 cube", Description: "Pocket cube", Aliases: []string{"2x2", "2x2x2"}, DefaultMethod: "random-state"}, Generators: []puzzle.Generator{Cube222{}}}
}

func inverseTokens(solution []gocube.Move) []string {
	if len(solution) == 0 {
		return []string{"R", "R'"}
	}
	result := make([]string, len(solution))
	for i, move := range solution {
		result[len(solution)-1-i] = move.Inverse().String()
	}
	return result
}
func (Cube333) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (puzzle.Generated, error) {
	if options.Length != nil {
		return puzzle.Generated{}, fmt.Errorf("random-state does not accept a fixed length")
	}
	target, err := sample333(ctx, rng)
	if err != nil {
		return puzzle.Generated{}, err
	}
	tables, err := load333(ctx)
	if err != nil {
		return puzzle.Generated{}, err
	}
	solution, err := solve333(ctx, target, tables)
	if err != nil {
		return puzzle.Generated{}, err
	}
	check := target
	for _, move := range solution {
		check.Move(move)
	}
	if !check.Solved() {
		return puzzle.Generated{}, fmt.Errorf("random-state solver failed verification")
	}
	return puzzle.Generated{Moves: inverseTokens(solution)}, nil
}
func (Cube222) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (puzzle.Generated, error) {
	if options.Length != nil {
		return puzzle.Generated{}, fmt.Errorf("random-state does not accept a fixed length")
	}
	target, err := sample222(ctx, rng)
	if err != nil {
		return puzzle.Generated{}, err
	}
	tables, err := load222(ctx)
	if err != nil {
		return puzzle.Generated{}, err
	}
	solution, err := solve222(ctx, target, tables)
	if err != nil {
		return puzzle.Generated{}, err
	}
	check := target
	for _, move := range solution {
		check.Move(move)
	}
	if !check.Solved() {
		return puzzle.Generated{}, fmt.Errorf("random-state 222 solver failed verification")
	}
	return puzzle.Generated{Moves: inverseTokens(solution)}, nil
}
