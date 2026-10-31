package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestInspectionRoundTrip(t *testing.T) {
	c := runner(t)
	var out, errout bytes.Buffer
	if n := c.Run(context.Background(), []string{"inspect", "--alg", "[R,U] (F2)2", "--json"}, &out, &errout); n != 0 {
		t.Fatal(n, errout.String())
	}
	var doc struct {
		Sequences []struct{ Sequence, Inverse, Facelets string }
	}
	if e := json.Unmarshal(out.Bytes(), &doc); e != nil {
		t.Fatal(e)
	}
	if doc.Sequences[0].Sequence != "R U R' U' F2 F2" || len(doc.Sequences[0].Facelets) != 54 {
		t.Fatal(doc)
	}
	c.Input = strings.NewReader(out.String())
	out.Reset()
	errout.Reset()
	if n := c.Run(context.Background(), []string{"inverse", "--stdin"}, &out, &errout); n != 0 {
		t.Fatal(n, errout.String())
	}
	if strings.TrimSpace(out.String()) != doc.Sequences[0].Inverse {
		t.Fatal(out.String(), doc)
	}
}
func TestSavedBatchInspection(t *testing.T) {
	c := runner(t)
	var out, errout bytes.Buffer
	if n := c.Run(context.Background(), []string{"--seed", "18446744073709551615", "--count", "2", "--json"}, &out, &errout); n != 0 {
		t.Fatal(n, errout.String())
	}
	c.Input = strings.NewReader(out.String())
	out.Reset()
	if n := c.Run(context.Background(), []string{"inspect", "--stdin", "--index", "2", "--json"}, &out, &errout); n != 0 {
		t.Fatal(n, errout.String())
	}
	if !strings.Contains(out.String(), `"seed": "18446744073709551615"`) || strings.Count(out.String(), `"facelets"`) != 1 {
		t.Fatal(out.String())
	}
}
func TestSequenceInvalidInputsNoOutput(t *testing.T) {
	for _, args := range [][]string{
		{"inspect", "--alg", "R X"}, {"inspect", "--alg", "M", "--puzzle", "222"},
		{"inspect", "--alg", "R", "--steps", "--json"}, {"inspect", "--alg", "R", "--alg", "U"},
		{"inspect", "--alg", "R", "U"}, {"inspect", "--alg", "R", "--index", "2"},
		{"inspect", "--alg", "R", "--color-depth", "bad"}, {"inspect", "--alg", "R", "--length", "2"},
		{"guide", "--alg", "R"},
	} {
		status, out, e := run(t, args...)
		if status != 2 || out != "" {
			t.Fatal(args, status, out, e)
		}
	}
}
func TestInspectionRejectsCorruptSavedBatch(t *testing.T) {
	for _, data := range []string{`{"schema_version":2}`, `{"schema_version":1,"kind":"unknown"}`, `{"schema_version":1,"puzzle":"333","parameters":{"count":1},"scrambles":[{"index":1,"moves":["R"],"sequence":"U","move_count":1}]}`} {
		c := runner(t)
		c.Input = strings.NewReader(data)
		var out, e bytes.Buffer
		if n := c.Run(context.Background(), []string{"inspect", "--stdin"}, &out, &e); n != 2 || out.Len() != 0 {
			t.Fatal(n, out.String(), e.String())
		}
	}
}
