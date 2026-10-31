package practice

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"
)

func (s *Store) ExportJSON(ctx context.Context, w io.Writer) error {
	doc, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(doc)
}

func (s *Store) ImportJSON(ctx context.Context, r io.Reader, mode ImportMode) (ImportSummary, error) {
	if err := ctx.Err(); err != nil {
		return ImportSummary{}, err
	}
	data, err := limitedRead(r)
	if err != nil {
		return ImportSummary{}, err
	}
	doc, err := decode(data)
	if err != nil {
		return ImportSummary{}, err
	}
	return s.importDocument(ctx, doc, mode)
}

func sameSolve(a, b Solve) bool {
	return a.ID == b.ID && a.CreatedAt.Equal(b.CreatedAt) && a.DurationNS == b.DurationNS && a.Penalty == b.Penalty && a.Notes == b.Notes && a.Scramble == b.Scramble && a.Puzzle == b.Puzzle && a.Method == b.Method && a.AlgorithmVersion == b.AlgorithmVersion && a.Seed == b.Seed
}

func (s *Store) importDocument(ctx context.Context, incoming Document, mode ImportMode) (ImportSummary, error) {
	var summary ImportSummary
	if mode == "" {
		mode = Merge
	}
	if mode != Merge && mode != Replace {
		return summary, fmt.Errorf("import mode must be merge or replace")
	}
	err := s.update(ctx, func(current *Document) (bool, error) {
		if mode == Replace {
			*current = incoming
			summary.SessionsAdded = len(incoming.Sessions)
			for _, session := range incoming.Sessions {
				summary.SolvesAdded += len(session.Solves)
			}
			return true, nil
		}
		byID, byName, solveOwners := map[string]int{}, map[string]string{}, map[string]string{}
		for i, session := range current.Sessions {
			byID[session.ID], byName[session.Name] = i, session.ID
			for _, solve := range session.Solves {
				solveOwners[solve.ID] = session.ID
			}
		}
		for _, session := range incoming.Sessions {
			i, exists := byID[session.ID]
			if !exists {
				if _, conflict := byName[session.Name]; conflict {
					return false, fmt.Errorf("%w: session name %q belongs to a different id", ErrConflict, session.Name)
				}
				for _, solve := range session.Solves {
					if _, conflict := solveOwners[solve.ID]; conflict {
						return false, fmt.Errorf("%w: solve id %q already belongs to another session", ErrConflict, solve.ID)
					}
					solveOwners[solve.ID] = session.ID
				}
				byID[session.ID], byName[session.Name] = len(current.Sessions), session.ID
				current.Sessions = append(current.Sessions, session)
				summary.SessionsAdded++
				summary.SolvesAdded += len(session.Solves)
				continue
			}
			existing := &current.Sessions[i]
			if existing.Name != session.Name || !existing.CreatedAt.Equal(session.CreatedAt) {
				return false, fmt.Errorf("%w: session %q metadata differs", ErrConflict, session.ID)
			}
			solves := map[string]Solve{}
			for _, solve := range existing.Solves {
				solves[solve.ID] = solve
			}
			for _, solve := range session.Solves {
				if previous, duplicate := solves[solve.ID]; duplicate {
					if !sameSolve(previous, solve) {
						return false, fmt.Errorf("%w: solve %q differs", ErrConflict, solve.ID)
					}
					summary.Duplicates++
					continue
				}
				if _, conflict := solveOwners[solve.ID]; conflict {
					return false, fmt.Errorf("%w: solve id %q already belongs to another session", ErrConflict, solve.ID)
				}
				existing.Solves = append(existing.Solves, solve)
				solves[solve.ID], solveOwners[solve.ID] = solve, session.ID
				summary.SolvesAdded++
			}
		}
		return summary.SessionsAdded > 0 || summary.SolvesAdded > 0, nil
	})
	if err != nil {
		return ImportSummary{}, err
	}
	return summary, nil
}

func csvHeader() []string {
	return []string{"schema_version", "kind", "session_id", "session_name", "session_created_at", "solve_id", "solve_created_at", "duration_ns", "penalty", "puzzle", "method", "algorithm_version", "seed", "scramble", "notes"}
}

// ExportCSV emits explicit session rows, including empty sessions, followed by
// their solve rows. Durations are decimal integer nanoseconds, never floats.
func (s *Store) ExportCSV(ctx context.Context, w io.Writer) error {
	doc, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	if err := writer.Write(csvHeader()); err != nil {
		return err
	}
	for _, session := range doc.Sessions {
		if err := ctx.Err(); err != nil {
			return err
		}
		row := make([]string, len(csvHeader()))
		row[0], row[1], row[2], row[3], row[4] = "1", "session", session.ID, session.Name, session.CreatedAt.Format(time.RFC3339Nano)
		if err := writer.Write(row); err != nil {
			return err
		}
		for _, solve := range session.Solves {
			if err := ctx.Err(); err != nil {
				return err
			}
			row := []string{"1", "solve", session.ID, session.Name, session.CreatedAt.Format(time.RFC3339Nano), solve.ID, solve.CreatedAt.Format(time.RFC3339Nano), strconv.FormatInt(solve.DurationNS, 10), string(solve.Penalty), solve.Puzzle, solve.Method, solve.AlgorithmVersion, solve.Seed, solve.Scramble, solve.Notes}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}
	writer.Flush()
	return writer.Error()
}

func (s *Store) ImportCSV(ctx context.Context, r io.Reader, mode ImportMode) (ImportSummary, error) {
	if err := ctx.Err(); err != nil {
		return ImportSummary{}, err
	}
	data, err := limitedRead(r)
	if err != nil {
		return ImportSummary{}, err
	}
	if !utf8.Valid(data) {
		return ImportSummary{}, fmt.Errorf("CSV must be valid UTF-8")
	}
	reader := csv.NewReader(bytes.NewReader(data))
	header, err := reader.Read()
	if err != nil {
		return ImportSummary{}, err
	}
	wanted := csvHeader()
	if len(header) != len(wanted) {
		return ImportSummary{}, fmt.Errorf("unexpected practice CSV header")
	}
	for i := range wanted {
		if header[i] != wanted[i] {
			return ImportSummary{}, fmt.Errorf("unexpected practice CSV header field %q", header[i])
		}
	}
	doc := Document{SchemaVersion: SchemaVersion, Sessions: []Session{}}
	byID := map[string]int{}
	for line := 2; ; line++ {
		if err := ctx.Err(); err != nil {
			return ImportSummary{}, err
		}
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ImportSummary{}, fmt.Errorf("CSV row %d: %w", line, err)
		}
		if len(row) != len(wanted) || row[0] != "1" {
			return ImportSummary{}, fmt.Errorf("CSV row %d: unsupported schema or column count", line)
		}
		created, err := time.Parse(time.RFC3339Nano, row[4])
		if err != nil {
			return ImportSummary{}, fmt.Errorf("CSV row %d: invalid session timestamp", line)
		}
		switch row[1] {
		case "session":
			if _, duplicate := byID[row[2]]; duplicate {
				return ImportSummary{}, fmt.Errorf("CSV row %d: duplicate session row", line)
			}
			for _, value := range row[5:] {
				if value != "" {
					return ImportSummary{}, fmt.Errorf("CSV row %d: solve fields on session row", line)
				}
			}
			byID[row[2]] = len(doc.Sessions)
			doc.Sessions = append(doc.Sessions, Session{ID: row[2], Name: row[3], CreatedAt: created, Solves: []Solve{}})
		case "solve":
			i, exists := byID[row[2]]
			if !exists || doc.Sessions[i].Name != row[3] || !doc.Sessions[i].CreatedAt.Equal(created) {
				return ImportSummary{}, fmt.Errorf("CSV row %d: solve must follow its matching session row", line)
			}
			solveTime, err := time.Parse(time.RFC3339Nano, row[6])
			if err != nil {
				return ImportSummary{}, fmt.Errorf("CSV row %d: invalid solve timestamp", line)
			}
			duration, err := strconv.ParseInt(row[7], 10, 64)
			if err != nil || strconv.FormatInt(duration, 10) != row[7] {
				return ImportSummary{}, fmt.Errorf("CSV row %d: invalid integer duration_ns", line)
			}
			solve := Solve{ID: row[5], CreatedAt: solveTime, DurationNS: duration, Penalty: Penalty(row[8]), Puzzle: row[9], Method: row[10], AlgorithmVersion: row[11], Seed: row[12], Scramble: row[13], Notes: row[14]}
			doc.Sessions[i].Solves = append(doc.Sessions[i].Solves, solve)
		default:
			return ImportSummary{}, fmt.Errorf("CSV row %d: unknown record kind %q", line, row[1])
		}
	}
	if err := validate(doc); err != nil {
		return ImportSummary{}, err
	}
	return s.importDocument(ctx, doc, mode)
}
