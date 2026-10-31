// Package practice provides optional, explicit local solve sessions.
// Generation does not depend on this package or access its storage.
package practice

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion = 1
const maxDataBytes = 64 << 20
const maxSessions = 1000
const maxSolves = 100000

type Penalty string

const (
	None    Penalty = "none"
	PlusTwo Penalty = "+2"
	DNF     Penalty = "DNF"
)

type Solve struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	DurationNS       int64     `json:"duration_ns,string"`
	Penalty          Penalty   `json:"penalty"`
	Notes            string    `json:"notes,omitempty"`
	Scramble         string    `json:"scramble"`
	Puzzle           string    `json:"puzzle,omitempty"`
	Method           string    `json:"method,omitempty"`
	AlgorithmVersion string    `json:"algorithm_version,omitempty"`
	Seed             string    `json:"seed,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Solves    []Solve   `json:"solves"`
}

type Document struct {
	SchemaVersion int       `json:"schema_version"`
	Sessions      []Session `json:"sessions"`
}

type Edit struct {
	DurationNS *int64
	Penalty    *Penalty
	Notes      *string
}

type ImportMode string

const (
	Merge   ImportMode = "merge"
	Replace ImportMode = "replace"
)

type ImportSummary struct {
	SessionsAdded int `json:"sessions_added"`
	SolvesAdded   int `json:"solves_added"`
	Duplicates    int `json:"duplicates"`
}

type Metric struct {
	Available  bool  `json:"available"`
	DNF        bool  `json:"dnf"`
	DurationNS int64 `json:"duration_ns,string"`
}

type Stats struct {
	Count    int    `json:"count"`
	DNFCount int    `json:"dnf_count"`
	Best     Metric `json:"best"`
	Worst    Metric `json:"worst"`
	Ao5      Metric `json:"ao5"`
	Ao12     Metric `json:"ao12"`
}

// Summarize returns current trailing ao5/ao12. Each drops the fastest and
// slowest result, treating DNF as worse than any time. Best/worst singles
// exclude DNFs when a valid time exists. Average nanoseconds are truncated
// only after division; storage never converts durations to floating point.
// Sources: WCA 9f8/9f9 and csTimer's default ceil(5%) trimming:
// https://www.worldcubeassociation.org/regulations/#9f8
// https://github.com/cs0x7f/cstimer/blob/master/src/js/stats/timestat.js
func Summarize(solves []Solve) Stats {
	stats := Stats{Count: len(solves)}
	for _, solve := range solves {
		if solve.Penalty == DNF {
			stats.DNFCount++
			continue
		}
		n := solve.DurationNS
		if solve.Penalty == PlusTwo {
			n += int64(2 * time.Second)
		}
		if !stats.Best.Available || n < stats.Best.DurationNS {
			stats.Best = Metric{Available: true, DurationNS: n}
		}
		if !stats.Worst.Available || n > stats.Worst.DurationNS {
			stats.Worst = Metric{Available: true, DurationNS: n}
		}
	}
	if len(solves) > 0 && !stats.Best.Available {
		stats.Best, stats.Worst = Metric{Available: true, DNF: true}, Metric{Available: true, DNF: true}
	}
	stats.Ao5 = average(solves, 5)
	stats.Ao12 = average(solves, 12)
	return stats
}

func average(solves []Solve, window int) Metric {
	if len(solves) < window {
		return Metric{}
	}
	values := make([]Metric, window)
	for i, solve := range solves[len(solves)-window:] {
		values[i] = Metric{Available: true, DNF: solve.Penalty == DNF, DurationNS: solve.DurationNS}
		if solve.Penalty == PlusTwo {
			values[i].DurationNS += int64(2 * time.Second)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].DNF != values[j].DNF {
			return !values[i].DNF
		}
		return values[i].DurationNS < values[j].DurationNS
	})
	var quotient, remainder int64
	n := int64(window - 2)
	for _, value := range values[1 : window-1] {
		if value.DNF {
			return Metric{Available: true, DNF: true}
		}
		// Dividing before summing avoids overflow even near MaxInt64.
		quotient += value.DurationNS / n
		remainder += value.DurationNS % n
	}
	return Metric{Available: true, DurationNS: quotient + remainder/n}
}

func validText(text string, limit int, multiline bool) bool {
	if len(text) > limit || !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		if unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\t')) {
			return false
		}
	}
	return true
}

func validID(id string) bool {
	if len(id) == 0 || len(id) > 80 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func validateSolve(s Solve) error {
	if !validID(s.ID) || s.CreatedAt.IsZero() || s.CreatedAt.Year() < 0 || s.CreatedAt.Year() > 9999 {
		return fmt.Errorf("solve must have a valid id and creation timestamp")
	}
	if s.Penalty != None && s.Penalty != PlusTwo && s.Penalty != DNF {
		return fmt.Errorf("solve %q: penalty must be none, +2, or DNF", s.ID)
	}
	if s.DurationNS < 0 || (s.DurationNS == 0 && s.Penalty != DNF) || s.DurationNS > math.MaxInt64-int64(2*time.Second) {
		return fmt.Errorf("solve %q: duration_ns must be positive and leave room for +2; zero is allowed for DNF", s.ID)
	}
	if strings.TrimSpace(s.Scramble) == "" || !validText(s.Scramble, 100000, false) {
		return fmt.Errorf("solve %q: scramble must be nonempty, bounded, and contain no control characters", s.ID)
	}
	if !validText(s.Notes, 16384, true) {
		return fmt.Errorf("solve %q: invalid notes", s.ID)
	}
	for _, value := range []string{s.Puzzle, s.Method, s.AlgorithmVersion} {
		if !validText(value, 256, false) {
			return fmt.Errorf("solve %q: invalid generation metadata", s.ID)
		}
	}
	if s.Seed != "" {
		n, err := strconv.ParseUint(s.Seed, 10, 64)
		if err != nil || strconv.FormatUint(n, 10) != s.Seed {
			return fmt.Errorf("solve %q: seed must be a canonical unsigned 64-bit decimal string", s.ID)
		}
	}
	return nil
}

func validate(doc Document) error {
	if doc.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported practice schema version %d", doc.SchemaVersion)
	}
	if doc.Sessions == nil || len(doc.Sessions) > maxSessions {
		return fmt.Errorf("sessions must be an array with at most %d entries", maxSessions)
	}
	ids, names, solveIDs := map[string]string{}, map[string]string{}, map[string]bool{}
	total := 0
	for _, session := range doc.Sessions {
		if !validID(session.ID) || session.CreatedAt.IsZero() || session.CreatedAt.Year() < 0 || session.CreatedAt.Year() > 9999 || strings.TrimSpace(session.Name) != session.Name || session.Name == "" || !validText(session.Name, 256, false) || session.Solves == nil {
			return fmt.Errorf("invalid session id, name, timestamp, or solves array")
		}
		if _, ok := ids[session.ID]; ok {
			return fmt.Errorf("duplicate session id %q", session.ID)
		}
		if _, ok := names[session.Name]; ok {
			return fmt.Errorf("duplicate session name %q", session.Name)
		}
		ids[session.ID], names[session.Name] = session.Name, session.ID
		total += len(session.Solves)
		if total > maxSolves {
			return fmt.Errorf("practice store supports at most %d solves", maxSolves)
		}
		for _, solve := range session.Solves {
			if err := validateSolve(solve); err != nil {
				return err
			}
			if solveIDs[solve.ID] {
				return fmt.Errorf("duplicate solve id %q", solve.ID)
			}
			solveIDs[solve.ID] = true
		}
	}
	for id := range ids {
		if owner, ok := names[id]; ok && owner != id {
			return fmt.Errorf("session name conflicts with another session id %q", id)
		}
	}
	return nil
}
