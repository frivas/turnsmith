package randomstate

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/unixpickle/gocube"
)

// Cube333WCA samples states satisfying the standard 3x3 minimum-distance rule.
// It is a practice generator, not an official WCA scramble program.
type Cube333WCA struct{}

// Cube222WCA samples rotation-equivalence classes at least four moves away.
type Cube222WCA struct{}

func (Cube333WCA) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "wca", AlgorithmVersion: "1", DirectEntropy: true, Description: "WCA-distance-filtered random-state 3×3 practice", Guarantees: "With uniform input words, uniform reachable-state sampling conditioned on optimal distance at least 2 outer-face moves; sequence reaches accepted target", Limitations: "Practice only; not an official WCA scramble program; standard 3×3 event only, no blindfolded orientation; bounded search can fail"}
}
func (Cube222WCA) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "wca", AlgorithmVersion: "1", DirectEntropy: true, Description: "WCA-distance-filtered random-state 2×2 practice", Guarantees: "With uniform input words, uniform rotation-equivalence-class sampling conditioned on optimal distance at least 4 face turns; sequence reaches accepted target", Limitations: "Practice only; not an official WCA scramble program; lazy tables and bounded sampling"}
}

const maximumWCASamples = 1024

// Every state at distance one is one of these 18 transitions from solved.
// Half turns count as one face turn. Comparing full cubie states avoids relying
// on the length of the nonoptimal two-phase solution returned by the solver.
func shallow333(target gocube.CubieCube) bool {
	if target.Solved() {
		return true
	}
	for move := gocube.Move(0); move < 18; move++ {
		state := gocube.SolvedCubieCube()
		state.Move(move)
		if state == target {
			return true
		}
	}
	return false
}

func sample333WCA(ctx context.Context, rng random.Source) (gocube.CubieCube, error) {
	for attempt := 0; attempt < maximumWCASamples; attempt++ {
		target, err := sample333(ctx, rng)
		if err != nil {
			return target, err
		}
		if !shallow333(target) {
			return target, nil
		}
	}
	return gocube.CubieCube{}, fmt.Errorf("wca sampler rejected %d near-solved targets", maximumWCASamples)
}
func sample222WCA(ctx context.Context, rng random.Source, tables *tables222) (gocube.CubieCorners, error) {
	for attempt := 0; attempt < maximumWCASamples; attempt++ {
		target, err := sample222(ctx, rng)
		if err != nil {
			return target, err
		}
		distance := tables.distance[permutationCode(target)*pocketOrientations+orientationCode(target)]
		if distance == 255 {
			return target, fmt.Errorf("sampled 222 state is not reachable")
		}
		if distance >= 4 {
			return target, nil
		}
	}
	return gocube.CubieCorners{}, fmt.Errorf("wca sampler rejected %d near-solved targets", maximumWCASamples)
}
func (Cube333WCA) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (puzzle.Generated, error) {
	if options.Length != nil {
		return puzzle.Generated{}, fmt.Errorf("wca does not accept a fixed length")
	}
	target, err := sample333WCA(ctx, rng)
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
		return puzzle.Generated{}, fmt.Errorf("wca 333 solver failed verification")
	}
	return puzzle.Generated{Moves: inverseTokens(solution)}, nil
}
func (Cube222WCA) Generate(ctx context.Context, rng random.Source, options puzzle.Options) (puzzle.Generated, error) {
	if options.Length != nil {
		return puzzle.Generated{}, fmt.Errorf("wca does not accept a fixed length")
	}
	tables, err := load222(ctx)
	if err != nil {
		return puzzle.Generated{}, err
	}
	target, err := sample222WCA(ctx, rng, tables)
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
		return puzzle.Generated{}, fmt.Errorf("wca 222 solver failed verification")
	}
	return puzzle.Generated{Moves: inverseTokens(solution)}, nil
}
