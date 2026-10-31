package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/frivas/turnsmith/internal/terminal"
	"github.com/frivas/turnsmith/internal/view"
	"io"
	"os"
	"strings"
)

type sequenceInput struct {
	Puzzle, Seed, Method, AlgorithmVersion, AppVersion string
	Sequences                                          [][]alg.Move
}
type inspectionRecord struct {
	Index     int      `json:"index"`
	Moves     []string `json:"moves"`
	Sequence  string   `json:"sequence"`
	MoveCount int      `json:"move_count"`
	Inverse   string   `json:"inverse"`
	Facelets  string   `json:"facelets"`
	Solved    bool     `json:"solved"`
}
type inspectionDocument struct {
	SchemaVersion    int                `json:"schema_version"`
	Kind             string             `json:"kind"`
	Puzzle           string             `json:"puzzle"`
	Seed             string             `json:"seed,omitempty"`
	Method           string             `json:"method,omitempty"`
	AlgorithmVersion string             `json:"algorithm_version,omitempty"`
	SourceAppVersion string             `json:"source_app_version,omitempty"`
	Sequences        []inspectionRecord `json:"sequences"`
}

func sequenceOptions(args []string) (commandOptions, error) {
	return options(args, map[string]string{"alg": "", "file": "", "stdin": "false", "index": "0", "puzzle": "", "steps": "false", "diagram": "false", "json": "false", "color": "auto", "color-depth": "auto", "seed": "", "method": "", "length": ""}, "stdin", "steps", "diagram", "json")
}
func (o commandOptions) hasSequence() bool {
	return o.seen["alg"] || o.seen["file"] || o.yes("stdin") || len(o.pos) > 0
}
func (c CLI) readSequences(o commandOptions) (sequenceInput, error) {
	count := 0
	var data []byte
	var err error
	if o.seen["alg"] {
		count++
		data = []byte(o.values["alg"])
	}
	if len(o.pos) > 0 {
		if len(o.pos) > 1 {
			return sequenceInput{}, app.Invalid("quote the whole algorithm as one argument")
		}
		count++
		data = []byte(o.pos[0])
	}
	if o.seen["file"] {
		count++
		f, e := os.Open(o.values["file"])
		if e != nil {
			return sequenceInput{}, e
		}
		defer f.Close()
		data, err = readBounded(f)
	}
	if o.yes("stdin") {
		count++
		data, err = readBounded(c.Input)
	}
	if count == 0 && c.Input != nil && !terminal.IsTerminal(c.Input) {
		count++
		data, err = readBounded(c.Input)
	}
	if count != 1 {
		return sequenceInput{}, app.Invalid("choose exactly one algorithm source: --alg, --file, --stdin, or a quoted argument")
	}
	if err != nil {
		return sequenceInput{}, err
	}
	result := sequenceInput{Puzzle: "333"}
	if strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		var envelope struct {
			SchemaVersion int             `json:"schema_version"`
			Kind          string          `json:"kind"`
			Result        json.RawMessage `json:"result"`
		}
		if e := json.Unmarshal(data, &envelope); e != nil {
			return result, invalid(e)
		}
		if envelope.SchemaVersion != 1 {
			return result, app.Invalid("unsupported input schema version %d", envelope.SchemaVersion)
		}
		if envelope.Kind == "practice-set" {
			data = envelope.Result
			if len(data) == 0 {
				return result, app.Invalid("practice set is missing its result")
			}
		}
		var batch app.Result
		if envelope.Kind == "inspection" {
			var doc inspectionDocument
			if e := json.Unmarshal(data, &doc); e != nil {
				return result, invalid(e)
			}
			batch.SchemaVersion = doc.SchemaVersion
			batch.Puzzle = doc.Puzzle
			batch.Seed = doc.Seed
			batch.Method = doc.Method
			batch.AlgorithmVersion = doc.AlgorithmVersion
			batch.AppVersion = doc.SourceAppVersion
			batch.Parameters.Count = len(doc.Sequences)
			for _, item := range doc.Sequences {
				batch.Scrambles = append(batch.Scrambles, app.Scramble{Index: item.Index, Moves: item.Moves, Sequence: item.Sequence, MoveCount: item.MoveCount})
			}
		} else {
			if envelope.Kind != "" && envelope.Kind != "practice-set" {
				return result, app.Invalid("unsupported JSON kind %q", envelope.Kind)
			}
			if e := json.Unmarshal(data, &batch); e != nil {
				return result, invalid(e)
			}
		}
		if batch.SchemaVersion != 1 || len(batch.Scrambles) == 0 || batch.Parameters.Count != len(batch.Scrambles) {
			return result, app.Invalid("invalid saved scramble batch or count")
		}
		if batch.Seed != "" {
			if _, e := random.ParseSeed(batch.Seed); e != nil {
				return result, invalid(e)
			}
		}
		result.Puzzle, result.Seed, result.Method, result.AlgorithmVersion, result.AppVersion = batch.Puzzle, batch.Seed, batch.Method, batch.AlgorithmVersion, batch.AppVersion
		total := 0
		for i, item := range batch.Scrambles {
			if item.Index != i+1 || item.MoveCount != len(item.Moves) || strings.Join(item.Moves, " ") != item.Sequence {
				return result, app.Invalid("saved scramble %d has inconsistent index/moves/sequence", i+1)
			}
			total += item.MoveCount
			if total > 10000 {
				return result, app.Invalid("saved input exceeds 10000 moves")
			}
			var moves []alg.Move
			for j, token := range item.Moves {
				m, e := alg.Parse(token, 1)
				if e != nil {
					return result, app.Invalid("scramble %d move %d: %v", i+1, j+1, e)
				}
				moves = append(moves, m...)
			}
			result.Sequences = append(result.Sequences, moves)
		}
	} else {
		moves, e := alg.Parse(string(data), 1000)
		if e != nil {
			return result, invalid(e)
		}
		result.Sequences = [][]alg.Move{moves}
		if o.values["puzzle"] != "" {
			result.Puzzle = o.values["puzzle"]
		}
	}
	size, e := cubestate.Size(result.Puzzle)
	if e != nil {
		return result, invalid(e)
	}
	result.Puzzle = fmt.Sprintf("%d%d%d", size, size, size)
	if o.values["puzzle"] != "" {
		requested, e := cubestate.Size(o.values["puzzle"])
		if e != nil || requested != size {
			return result, app.Invalid("--puzzle does not match the saved input")
		}
	}
	index, e := o.number("index", 0, 10000)
	if e != nil {
		return result, e
	}
	if index > len(result.Sequences) {
		return result, app.Invalid("--index exceeds the saved batch size")
	}
	if index > 0 {
		result.Sequences = result.Sequences[index-1 : index]
	}
	return result, nil
}
func (c CLI) viewOptions(o commandOptions) (view.Options, error) {
	v := view.Options{Color: c.Color, Depth: c.ColorDepth}
	switch o.values["color"] {
	case "auto":
	case "always":
		v.Color = true
	case "never":
		v.Color = false
	default:
		return v, app.Invalid("--color must be auto, always, or never")
	}
	switch o.values["color-depth"] {
	case "auto":
	case "16", "256", "truecolor":
		v.Depth = o.values["color-depth"]
	default:
		return v, app.Invalid("--color-depth must be auto, 16, 256, or truecolor")
	}
	if v.Depth == "" {
		v.Depth = "256"
	}
	return v, nil
}
func (c CLI) generatedSequence(ctx context.Context, o commandOptions) (sequenceInput, error) {
	req := app.Request{Puzzle: o.values["puzzle"], Method: o.values["method"], Count: 1}
	if req.Puzzle == "" {
		req.Puzzle = "333"
	}
	if o.seen["seed"] {
		seed, e := random.ParseSeed(o.values["seed"])
		if e != nil {
			return sequenceInput{}, invalid(e)
		}
		req.Seed = &seed
	}
	if o.seen["length"] {
		n, e := integer(o.values["length"], "length", 1000)
		if e != nil {
			return sequenceInput{}, e
		}
		req.Length = &n
	}
	result, e := c.Service.Generate(ctx, req)
	if e != nil {
		return sequenceInput{}, e
	}
	moves, e := alg.Parse(result.Scrambles[0].Sequence, 1000)
	if e != nil {
		return sequenceInput{}, e
	}
	return sequenceInput{Puzzle: result.Puzzle, Seed: result.Seed, Method: result.Method, AlgorithmVersion: result.AlgorithmVersion, AppVersion: result.AppVersion, Sequences: [][]alg.Move{moves}}, nil
}
func (c CLI) sequenceCommand(ctx context.Context, name string, args []string, w io.Writer) error {
	o, e := sequenceOptions(args)
	if e != nil {
		return e
	}
	if o.yes("help") {
		return commandHelp(w, name)
	}
	if o.yes("steps") && o.yes("diagram") {
		return app.Invalid("choose --steps or --diagram")
	}
	if o.yes("json") && (o.yes("steps") || o.yes("diagram")) {
		return app.Invalid("JSON cannot be combined with diagram options")
	}
	if name != "guide" && (o.seen["seed"] || o.seen["length"] || o.seen["method"]) {
		return app.Invalid("generation flags are only available with guide")
	}
	if name == "guide" {
		if o.yes("json") || o.yes("steps") || o.yes("diagram") {
			return app.Invalid("guide is interactive; use inspect for streamed output")
		}
		if e = c.checkTerminal(w); e != nil {
			return e
		}
	}
	var input sequenceInput
	if name == "guide" && !o.hasSequence() {
		input, e = c.generatedSequence(ctx, o)
	} else {
		if o.hasSequence() && (o.seen["seed"] || o.seen["length"] || o.seen["method"]) {
			return app.Invalid("supplied moves cannot be combined with generation flags")
		}
		input, e = c.readSequences(o)
	}
	if e != nil {
		return e
	}
	style, e := c.viewOptions(o)
	if e != nil {
		return e
	}
	size, _ := cubestate.Size(input.Puzzle)
	if name == "guide" {
		if len(input.Sequences) != 1 {
			return app.Invalid("interactive guide needs one sequence; select --index N")
		}
		moves := input.Sequences[0]
		states, e := cubestate.Replay(size, moves)
		if e != nil {
			return invalid(e)
		}
		return terminal.RunGuideWithOptions(ctx, c.Input, w, len(moves), terminal.GuideOptions{MinWidth: 2*max(10, size*3) + 2, MinHeight: 3*(size+1) + 6}, func(index, width, height int, action string) (string, error) {
			return view.Frame(states, moves, index, width, height, action, style), nil
		})
	}
	doc := inspectionDocument{SchemaVersion: 1, Kind: "inspection", Puzzle: input.Puzzle, Seed: input.Seed, Method: input.Method, AlgorithmVersion: input.AlgorithmVersion, SourceAppVersion: input.AppVersion}
	for i, moves := range input.Sequences {
		if e := ctx.Err(); e != nil {
			return e
		}
		states, e := cubestate.Replay(size, moves)
		if e != nil {
			return invalid(e)
		}
		final := states[len(states)-1]
		doc.Sequences = append(doc.Sequences, inspectionRecord{Index: i + 1, Moves: alg.Tokens(moves), Sequence: alg.Format(moves), MoveCount: len(moves), Inverse: alg.Format(alg.Inverse(moves)), Facelets: final.Facelets(), Solved: final.Solved()})
	}
	if o.yes("json") {
		if name == "inverse" {
			for i := range doc.Sequences {
				moves, _ := alg.Parse(doc.Sequences[i].Inverse, 1000)
				states, _ := cubestate.Replay(size, moves)
				state := states[len(states)-1]
				doc.Sequences[i] = inspectionRecord{Index: i + 1, Moves: alg.Tokens(moves), Sequence: alg.Format(moves), MoveCount: len(moves), Inverse: alg.Format(alg.Inverse(moves)), Facelets: state.Facelets(), Solved: state.Solved()}
			}
		}
		return writeJSON(w, doc)
	}
	for i, record := range doc.Sequences {
		if name == "inverse" {
			if _, e = fmt.Fprintln(w, record.Inverse); e != nil {
				return e
			}
			continue
		}
		if _, e = fmt.Fprintf(w, "Sequence %d/%d (%s): %s\nInverse: %s\n", i+1, len(doc.Sequences), input.Puzzle, record.Sequence, record.Inverse); e != nil {
			return e
		}
		if o.yes("steps") {
			e = (view.Guide{Size: size, Depth: style.Depth}).Render(ctx, w, record.Moves, style.Color)
		} else {
			moves := input.Sequences[i]
			states, _ := cubestate.Replay(size, moves)
			_, e = io.WriteString(w, view.Orientation(size)+view.Net(states[len(states)-1], style))
		}
		if e != nil {
			return fmt.Errorf("write output: %w", e)
		}
	}
	return nil
}
