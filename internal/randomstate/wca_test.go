package randomstate

import (
	"context"
	"errors"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/unixpickle/gocube"
	"reflect"
	"strings"
	"testing"
)

type prefixedSource struct {
	words    []uint64
	fallback random.Source
	calls    int
}

func (s *prefixedSource) Uint64() uint64 {
	s.calls++
	if len(s.words) > 0 {
		value := s.words[0]
		s.words = s.words[1:]
		return value
	}
	return s.fallback.Uint64()
}

type repeatingSource struct {
	words []uint64
	calls int
}

func (s *repeatingSource) Uint64() uint64 {
	value := s.words[s.calls%len(s.words)]
	s.calls++
	return value
}
func solvedWords(size int) []uint64 {
	out := []uint64{}
	if size == 3 {
		for i := 7; i > 0; i-- {
			out = append(out, uint64(i))
		}
		for i := 11; i > 0; i-- {
			out = append(out, uint64(i))
		}
		for i := 0; i < 7; i++ {
			out = append(out, 1)
		}
		for i := 0; i < 11; i++ {
			out = append(out, 0)
		}
	} else {
		for i := 6; i > 0; i-- {
			out = append(out, uint64(i))
		}
		for i := 0; i < 6; i++ {
			out = append(out, 1)
		}
	}
	return out
}
func TestWCARejectsNearSolvedBeforeSolving(t *testing.T) {
	ctx := context.Background()
	tables, _ := load222(ctx)
	for _, size := range []int{2, 3} {
		seed := func() random.Source { return random.Derive(0, "fixture", "wca", "1", 0) }
		prefix := solvedWords(size)
		stream := &prefixedSource{words: append([]uint64{}, prefix...), fallback: seed()}
		if size == 2 {
			solved, e := sample222(ctx, &repeatingSource{words: prefix})
			if e != nil || !solved.Solved() {
				t.Fatal("script is not solved", e)
			}
			expected, e := sample222(ctx, seed())
			if e != nil {
				t.Fatal(e)
			}
			actual, e := sample222WCA(ctx, stream, tables)
			if e != nil || actual != expected {
				t.Fatal("222 rejected wrong target", e)
			}
		} else {
			solved, e := sample333(ctx, &repeatingSource{words: prefix})
			if e != nil || !solved.Solved() {
				t.Fatal("script is not solved", e)
			}
			expected, e := sample333(ctx, seed())
			if e != nil {
				t.Fatal(e)
			}
			actual, e := sample333WCA(ctx, stream)
			if e != nil || actual != expected {
				t.Fatal("333 rejected wrong target", e)
			}
		}
		if stream.calls <= len(prefix) {
			t.Fatal("near-solved sample not rejected")
		}
	}
}
func TestWCASamplingBoundAndCancellation(t *testing.T) {
	ctx := context.Background()
	tables, _ := load222(ctx)
	for _, size := range []int{2, 3} {
		stream := &repeatingSource{words: solvedWords(size)}
		var err error
		if size == 2 {
			_, err = sample222WCA(ctx, stream, tables)
		} else {
			_, err = sample333WCA(ctx, stream)
		}
		if err == nil || stream.calls != maximumWCASamples*len(stream.words) {
			t.Fatal("unbounded rejection", size, stream.calls, err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, g := range []puzzle.Generator{Cube222WCA{}, Cube333WCA{}} {
		_, err := g.Generate(cancelled, random.Derive(0, "x", "wca", "1", 0), puzzle.Options{})
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		n := 20
		if _, err := g.Generate(ctx, random.Derive(0, "x", "wca", "1", 0), puzzle.Options{Length: &n}); err == nil {
			t.Fatal("accepted length")
		}
	}
}
func TestWCA333ShallowSet(t *testing.T) {
	solved := gocube.SolvedCubieCube()
	excluded := map[gocube.CubieCube]bool{solved: true}
	for move := gocube.Move(0); move < 18; move++ {
		state := solved
		state.Move(move)
		excluded[state] = true
		if !shallow333(state) {
			t.Fatal("one move accepted", move)
		}
	}
	if len(excluded) != 19 || !shallow333(solved) {
		t.Fatal("invalid shallow set")
	}
	for a := gocube.Move(0); a < 18; a++ {
		for b := gocube.Move(0); b < 18; b++ {
			state := solved
			state.Move(a)
			state.Move(b)
			if shallow333(state) != excluded[state] {
				t.Fatal("incorrect distance-two classification", a, b)
			}
		}
	}
}

func rotationPaths(t *testing.T) [][]alg.Move {
	t.Helper()
	solved, _ := cubestate.New(2)
	paths := [][]alg.Move{{}}
	states := []cubestate.State{solved}
	seen := map[string]bool{solved.Facelets(): true}
	for front := 0; front < len(states); front++ {
		for _, face := range []byte{'x', 'y', 'z'} {
			move := alg.Move{Face: face, Width: 1, Turns: 1}
			next := states[front].Clone()
			if e := next.Apply(move); e != nil {
				t.Fatal(e)
			}
			key := next.Facelets()
			if !seen[key] {
				seen[key] = true
				states = append(states, next)
				paths = append(paths, append(append([]alg.Move{}, paths[front]...), move))
			}
		}
	}
	if len(paths) != 24 {
		t.Fatal("rotation group", len(paths))
	}
	return paths
}
func canonicalPocket(state cubestate.State, rotations [][]alg.Move) string {
	best := ""
	for _, path := range rotations {
		next := state.Clone()
		for _, move := range path {
			_ = next.Apply(move)
		}
		key := next.Facelets()
		if best == "" || key < best {
			best = key
		}
	}
	return best
}
func independentState(target gocube.CubieCube, size int) cubestate.State {
	state, _ := cubestate.New(size)
	stickers := target.StickerCube()
	faces := [6]cube.Face{cube.U, cube.D, cube.F, cube.B, cube.R, cube.L}
	for f, face := range faces {
		for i := range state.Faces[face] {
			source := i
			if size == 2 {
				source = []int{0, 2, 6, 8}[i]
			}
			state.Faces[face][i] = faces[stickers[f*9+source]-1]
		}
	}
	return state
}
func TestWCA222ThresholdIndependentAllFaceSearch(t *testing.T) {
	rotations := rotationPaths(t)
	solved, _ := cubestate.New(2)
	states := []cubestate.State{solved}
	depths := []int{0}
	excluded := map[string]bool{canonicalPocket(solved, rotations): true}
	for front := 0; front < len(states); front++ {
		if depths[front] == 3 {
			continue
		}
		for _, face := range []byte("UDLRFB") {
			for turns := 1; turns <= 3; turns++ {
				next := states[front].Clone()
				if e := next.Apply(alg.Move{Face: face, Width: 1, Turns: turns}); e != nil {
					t.Fatal(e)
				}
				key := canonicalPocket(next, rotations)
				if !excluded[key] {
					excluded[key] = true
					states = append(states, next)
					depths = append(depths, depths[front]+1)
				}
			}
		}
	}
	if len(excluded) != 385 {
		t.Fatal("independent shallow count", len(excluded))
	}
	tables, _ := load222(context.Background())
	count := 0
	for hash, distance := range tables.distance {
		if distance >= 4 {
			continue
		}
		count++
		corners := decodePermutation(hash / pocketOrientations)
		orientation := decodeOrientation(hash % pocketOrientations)
		for i := range corners {
			corners[i].Orientation = orientation[i].Orientation
		}
		target := gocube.SolvedCubieCube()
		target.Corners = corners
		key := canonicalPocket(independentState(target, 2), rotations)
		if !excluded[key] {
			t.Fatal("UFR threshold differs from independent six-face quotient", hash)
		}
	}
	if count != len(excluded) {
		t.Fatal("shallow set differs", count)
	}
}
func TestWCATargetsReachedAndDeterministic(t *testing.T) {
	ctx := context.Background()
	tables, _ := load222(ctx)
	for _, id := range []string{"222", "333"} {
		for seed := uint64(0); seed < 20; seed++ {
			source := func() random.Source { return random.Derive(seed, id, "wca", "1", 0) }
			target := gocube.SolvedCubieCube()
			var g puzzle.Generator
			var err error
			size := 3
			if id == "222" {
				size = 2
				target.Corners, err = sample222WCA(ctx, source(), tables)
				g = Cube222WCA{}
			} else {
				target, err = sample333WCA(ctx, source())
				g = Cube333WCA{}
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := g.Generate(ctx, source(), puzzle.Options{})
			if err != nil {
				t.Fatal(id, seed, err)
			}
			moves, err := alg.Parse(strings.Join(got.Moves, " "), 100)
			if err != nil {
				t.Fatal(err)
			}
			states, err := cubestate.Replay(size, moves)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(states[len(states)-1], independentState(target, size)) {
				t.Fatal("independent target mismatch", id, seed)
			}
			if seed == 0 {
				again, err := g.Generate(ctx, source(), puzzle.Options{})
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("not deterministic", err)
				}
				t.Log(id, strings.Join(got.Moves, " "))
			}
		}
	}
}

func TestWCAGoldenSequences(t *testing.T) {
	for _, tc := range []struct {
		id, want string
		g        puzzle.Generator
	}{
		{"222", "F R' U2 R F R2 U F'", Cube222WCA{}},
		{"333", "L' U2 R F2 D2 R F2 L' B2 U2 R' U2 L B2 F2 B' R F2 U B F L' R'", Cube333WCA{}},
	} {
		got, err := tc.g.Generate(context.Background(), random.Derive(0, tc.id, "wca", "1", 0), puzzle.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got.Moves, " ") != tc.want {
			t.Fatal("wca algorithm1 changed", tc.id, got.Moves)
		}
		descriptor := tc.g.Descriptor()
		if !descriptor.DirectEntropy || descriptor.ID != "wca" || descriptor.AlgorithmVersion != "1" {
			t.Fatal("incorrect capabilities", descriptor)
		}
	}
}

func shuffleWords(target []int) []uint64 {
	current := make([]int, len(target))
	for i := range current {
		current[i] = i
	}
	words := []uint64{}
	for i := len(current) - 1; i > 0; i-- {
		j := 0
		for current[j] != target[i] {
			j++
		}
		words = append(words, uint64(j+i+1))
		current[i], current[j] = current[j], current[i]
	}
	return words
}
func tritWord(value int) uint64 {
	if value == 0 {
		return 3
	}
	return uint64(value)
}
func pocketWords(target gocube.CubieCorners) []uint64 {
	permutation := make([]int, 7)
	for i := range permutation {
		permutation[i] = target[i+1].Piece - 1
	}
	words := shuffleWords(permutation)
	for i := 1; i < 7; i++ {
		words = append(words, tritWord(target[i].Orientation))
	}
	return words
}
func fullWords(target gocube.CubieCube) []uint64 {
	corners := make([]int, 8)
	edges := make([]int, 12)
	for i := range corners {
		corners[i] = target.Corners[i].Piece
	}
	for i := range edges {
		edges[i] = target.Edges[i].Piece
	}
	words := append(shuffleWords(corners), shuffleWords(edges)...)
	for i := 0; i < 7; i++ {
		words = append(words, tritWord(target.Corners[i].Orientation))
	}
	for i := 0; i < 11; i++ {
		word := uint64(0)
		if target.Edges[i].Flip {
			word = 1
		}
		words = append(words, word)
	}
	return words
}
func TestWCARejectsEveryShallowState(t *testing.T) {
	ctx := context.Background()
	source := func() random.Source { return random.Derive(0, "fixture", "wca", "1", 0) }
	tables, _ := load222(ctx)
	expected222, _ := sample222(ctx, source())
	expected333, _ := sample333(ctx, source())
	for hash, distance := range tables.distance {
		if distance >= 4 {
			continue
		}
		target := decodePermutation(hash / pocketOrientations)
		orientation := decodeOrientation(hash % pocketOrientations)
		for i := range target {
			target[i].Orientation = orientation[i].Orientation
		}
		words := pocketWords(target)
		drawn, err := sample222(ctx, &prefixedSource{words: append([]uint64{}, words...), fallback: source()})
		if err != nil || drawn != target {
			t.Fatal("invalid scripted pocket target", hash, err)
		}
		actual, err := sample222WCA(ctx, &prefixedSource{words: words, fallback: source()}, tables)
		if err != nil || actual != expected222 {
			t.Fatal("accepted shallow pocket state", hash, distance, err)
		}
	}
	solved := gocube.SolvedCubieCube()
	for move := -1; move < 18; move++ {
		target := solved
		if move >= 0 {
			target.Move(gocube.Move(move))
		}
		words := fullWords(target)
		drawn, err := sample333(ctx, &prefixedSource{words: append([]uint64{}, words...), fallback: source()})
		if err != nil || drawn != target {
			t.Fatal("invalid scripted full target", move, err)
		}
		actual, err := sample333WCA(ctx, &prefixedSource{words: words, fallback: source()})
		if err != nil || actual != expected333 {
			t.Fatal("accepted shallow full state", move, err)
		}
	}
}

type cancellationSource struct {
	source    random.Source
	calls, at int
	cancel    context.CancelFunc
}

func (s *cancellationSource) Uint64() uint64 {
	s.calls++
	value := s.source.Uint64()
	if s.calls == s.at {
		s.cancel()
	}
	return value
}
func TestWCACancelledDuringRejection(t *testing.T) {
	for _, size := range []int{2, 3} {
		ctx, cancel := context.WithCancel(context.Background())
		words := solvedWords(size)
		source := &cancellationSource{source: &repeatingSource{words: words}, at: len(words), cancel: cancel}
		var g puzzle.Generator = Cube333WCA{}
		if size == 2 {
			g = Cube222WCA{}
		}
		_, err := g.Generate(ctx, source, puzzle.Options{})
		cancel()
		if !errors.Is(err, context.Canceled) || source.calls != len(words) {
			t.Fatal("cancelled rejection kept reading entropy", size, source.calls, err)
		}
	}
}
