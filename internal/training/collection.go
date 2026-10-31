// Package training manages user-owned algorithm collections and local progress.
package training

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cubestate"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxBytes = 16 << 20

type Collection struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	Cases         []Case `json:"cases"`
}
type Case struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Puzzle    string `json:"puzzle"`
	Algorithm string `json:"algorithm"`
}

func textOK(s string, max int) bool {
	return strings.TrimSpace(s) != "" && len(s) <= max && !strings.ContainsFunc(s, unicode.IsControl)
}
func (c Case) Size() (int, error) {
	switch c.Puzzle {
	case "222":
		return 2, nil
	case "333":
		return 3, nil
	case "444":
		return 4, nil
	}
	return 0, fmt.Errorf("unsupported case puzzle %q", c.Puzzle)
}
func (c Case) Setup() ([]alg.Move, error) {
	moves, e := alg.Parse(c.Algorithm, 10000)
	if e != nil {
		return nil, e
	}
	return alg.Inverse(moves), nil
}
func (c Collection) Validate() error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("unsupported collection schema version %d", c.SchemaVersion)
	}
	if !textOK(c.Name, 200) || len(c.Cases) < 1 || len(c.Cases) > 10000 {
		return fmt.Errorf("collection needs a name and 1–10000 cases")
	}
	ids := map[string]bool{}
	totalMoves := 0
	for _, item := range c.Cases {
		if !textOK(item.ID, 200) || !textOK(item.Name, 200) || ids[item.ID] {
			return fmt.Errorf("invalid or duplicate case ID %q", item.ID)
		}
		ids[item.ID] = true
		size, e := item.Size()
		if e != nil {
			return e
		}
		moves, e := alg.Parse(item.Algorithm, 10000)
		if e != nil {
			return fmt.Errorf("case %s: %w", item.ID, e)
		}
		totalMoves += len(moves)
		if totalMoves > 100000 {
			return fmt.Errorf("collection exceeds 100000 expanded moves")
		}
		setup := alg.Inverse(moves)
		joined := append(setup, moves...)
		states, e := cubestate.Replay(size, joined)
		if e != nil {
			return fmt.Errorf("case %s: %w", item.ID, e)
		}
		if !states[len(states)-1].Solved() {
			return fmt.Errorf("case %s setup verification failed", item.ID)
		}
	}
	return nil
}
func (c Collection) Fingerprint() (string, error) {
	if e := c.Validate(); e != nil {
		return "", e
	}
	data, _ := json.Marshal(c)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
func readBounded(r io.Reader) ([]byte, error) {
	data, e := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if e != nil {
		return nil, e
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("training data exceeds 16 MiB")
	}
	return data, nil
}
func decode(data []byte, out any) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("training JSON must be valid UTF-8")
	}
	// Reject duplicate object keys before the typed decode.
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, e := d.Token()
				if e != nil {
					return e
				}
				k := key.(string)
				if seen[k] {
					return fmt.Errorf("duplicate JSON key %q", k)
				}
				seen[k] = true
				if e := walk(); e != nil {
					return e
				}
			}
			_, e = d.Token()
			return e
		case '[':
			for d.More() {
				if e := walk(); e != nil {
					return e
				}
			}
			_, e = d.Token()
			return e
		}
		return fmt.Errorf("unexpected delimiter")
	}
	if e := walk(); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(out); e != nil {
		return e
	}
	return nil
}
func LoadCollection(r io.Reader) (Collection, error) {
	data, e := readBounded(r)
	if e != nil {
		return Collection{}, e
	}
	var c Collection
	if e := decode(data, &c); e != nil {
		return c, e
	}
	return c, c.Validate()
}
