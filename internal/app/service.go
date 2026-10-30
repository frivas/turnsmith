// Package app validates requests and assembles completed, reproducible batches.
package app

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
)

type InvalidInput struct{ Message string }

func (e *InvalidInput) Error() string { return e.Message }
func Invalid(format string, args ...any) error {
	return &InvalidInput{Message: fmt.Sprintf(format, args...)}
}

type Request struct {
	Puzzle string
	Method string
	Length *int
	Count  int
	Seed   *uint64
}
type Parameters struct {
	Length *int `json:"length,omitempty"`
	Count  int  `json:"count"`
}
type Scramble struct {
	Index     int      `json:"index"`
	Moves     []string `json:"moves"`
	Sequence  string   `json:"sequence"`
	MoveCount int      `json:"move_count"`
}
type Result struct {
	SchemaVersion    int        `json:"schema_version"`
	AppVersion       string     `json:"app_version"`
	Puzzle           string     `json:"puzzle"`
	Method           string     `json:"method"`
	AlgorithmVersion string     `json:"algorithm_version"`
	Seed             string     `json:"seed"`
	Parameters       Parameters `json:"parameters"`
	Scrambles        []Scramble `json:"scrambles"`
}
type Service struct {
	Registry *puzzle.Registry
	Entropy  io.Reader
	Version  string
}

func (s Service) Generate(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if req.Puzzle == "" {
		req.Puzzle = "333"
	}
	d, ok := s.Registry.Resolve(req.Puzzle)
	if !ok {
		return Result{}, Invalid("unsupported puzzle %q; run \"turnsmith puzzles\" to list supported puzzles", req.Puzzle)
	}
	if req.Method == "" {
		req.Method = d.DefaultMethod
	}
	g, ok := s.Registry.Generator(d.ID, req.Method)
	if !ok {
		return Result{}, Invalid("puzzle %q does not support method %q", d.ID, req.Method)
	}
	m := g.Descriptor()
	if req.Count < 1 || req.Count > 10000 {
		return Result{}, Invalid("--count must be an integer between 1 and 10000")
	}
	var length *int
	if !m.ExplicitLength {
		if req.Length != nil {
			return Result{}, Invalid("method %q does not support --length", m.ID)
		}
	} else {
		n := *m.DefaultLength
		if req.Length != nil {
			n = *req.Length
		}
		if n < 1 || n > 1000 || n < m.MinLength || n > m.MaxLength {
			return Result{}, Invalid("--length must be an integer between %d and %d", m.MinLength, m.MaxLength)
		}
		if n > 1000000/req.Count {
			return Result{}, Invalid("request exceeds the limit of 1000000 total moves")
		}
		length = &n
	}
	var seed uint64
	if req.Seed != nil {
		seed = *req.Seed
	} else {
		var err error
		seed, err = random.Seed(s.Entropy)
		if err != nil {
			return Result{}, err
		}
	}
	result := Result{SchemaVersion: 1, AppVersion: s.Version, Puzzle: d.ID, Method: m.ID, AlgorithmVersion: m.AlgorithmVersion, Seed: strconv.FormatUint(seed, 10), Parameters: Parameters{Length: length, Count: req.Count}, Scrambles: make([]Scramble, 0, req.Count)}
	total := 0
	for i := 0; i < req.Count; i++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		// Each call owns its options and stream, including when generators mutate inputs.
		options := puzzle.Options{}
		if length != nil {
			n := *length
			options.Length = &n
		}
		generated, err := g.Generate(ctx, random.Derive(seed, d.ID, m.ID, m.AlgorithmVersion, uint64(i)), options)
		if err != nil {
			return Result{}, fmt.Errorf("generate scramble %d: %w", i+1, err)
		}
		if length != nil && len(generated.Moves) != *length {
			return Result{}, fmt.Errorf("generate scramble %d: incorrect token count", i+1)
		}
		if len(generated.Moves) == 0 || len(generated.Moves) > 1000000-total {
			return Result{}, fmt.Errorf("generate scramble %d: invalid output size", i+1)
		}
		for _, token := range generated.Moves {
			if token == "" || strings.ContainsAny(token, " \t\r\n") {
				return Result{}, fmt.Errorf("generate scramble %d: invalid notation token", i+1)
			}
		}
		tokens := append([]string(nil), generated.Moves...)
		total += len(tokens)
		result.Scrambles = append(result.Scrambles, Scramble{Index: i + 1, Moves: tokens, Sequence: strings.Join(tokens, " "), MoveCount: len(tokens)})
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return result, nil
}
