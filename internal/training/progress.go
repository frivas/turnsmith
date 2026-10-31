package training

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/random"
	"time"
)

func initial(c Collection, fingerprint string) CollectionProgress {
	p := CollectionProgress{CollectionID: fingerprint, Name: c.Name, Cases: make([]CaseProgress, len(c.Cases))}
	for i, item := range c.Cases {
		p.Cases[i] = CaseProgress{CaseID: item.ID, Status: New, Attempts: []Attempt{}}
	}
	return p
}
func find(d *Document, c Collection, fingerprint string) *CollectionProgress {
	for i := range d.Collections {
		if d.Collections[i].CollectionID == fingerprint {
			return &d.Collections[i]
		}
	}
	d.Collections = append(d.Collections, initial(c, fingerprint))
	return &d.Collections[len(d.Collections)-1]
}
func (s *Store) Progress(ctx context.Context, c Collection) ([]CaseProgress, error) {
	fingerprint, e := c.Fingerprint()
	if e != nil {
		return nil, e
	}
	d, e := s.snapshot(ctx)
	if e != nil {
		return nil, e
	}
	for _, p := range d.Collections {
		if p.CollectionID == fingerprint {
			if len(p.Cases) != len(c.Cases) {
				return nil, fmt.Errorf("%w: collection case count differs", ErrCorrupt)
			}
			out := make([]CaseProgress, len(c.Cases))
			byID := map[string]CaseProgress{}
			for _, item := range p.Cases {
				byID[item.CaseID] = item
			}
			for i, item := range c.Cases {
				value, ok := byID[item.ID]
				if !ok {
					return nil, fmt.Errorf("%w: case missing", ErrCorrupt)
				}
				out[i] = value
			}
			return out, nil
		}
	}
	return initial(c, fingerprint).Cases, nil
}
func caseIndex(c Collection, id string) (int, error) {
	for i, item := range c.Cases {
		if item.ID == id {
			return i, nil
		}
	}
	return 0, fmt.Errorf("case %q not found", id)
}
func (s *Store) SetStatus(ctx context.Context, c Collection, caseID string, status Status) error {
	if !validStatus(status) {
		return fmt.Errorf("status must be new, learning or known")
	}
	fingerprint, e := c.Fingerprint()
	if e != nil {
		return e
	}
	if _, e := caseIndex(c, caseID); e != nil {
		return e
	}
	return s.update(ctx, func(d *Document) error {
		progress := find(d, c, fingerprint)
		for i := range progress.Cases {
			if progress.Cases[i].CaseID == caseID {
				progress.Cases[i].Status = status
				return nil
			}
		}
		return ErrCorrupt
	})
}

// Record appends an attempt without changing the user's explicit learning status.
// Empty attempt IDs and timestamps are assigned locally. Repeating an identical
// caller-provided ID is idempotent; changing its content is a conflict.
func (s *Store) Record(ctx context.Context, c Collection, caseID string, attempt Attempt) error {
	fingerprint, e := c.Fingerprint()
	if e != nil {
		return e
	}
	if _, e := caseIndex(c, caseID); e != nil {
		return e
	}
	if attempt.ID == "" {
		attempt.ID, e = id()
		if e != nil {
			return e
		}
	}
	if attempt.At.IsZero() {
		attempt.At = time.Now().UTC()
	}
	return s.update(ctx, func(d *Document) error {
		progress := find(d, c, fingerprint)
		for i := range progress.Cases {
			p := &progress.Cases[i]
			if p.CaseID != caseID {
				continue
			}
			for _, a := range p.Attempts {
				if a.ID == attempt.ID {
					if a.At.Equal(attempt.At) && a.DurationNS == attempt.DurationNS && a.Hint == attempt.Hint && a.Success == attempt.Success {
						return nil
					}
					return ErrConflict
				}
			}
			p.Attempts = append(p.Attempts, attempt)
			return nil
		}
		return ErrCorrupt
	})
}
func bounded(ctx context.Context, rng random.Source, n uint64) (int, error) {
	threshold := -n % n
	for {
		if e := ctx.Err(); e != nil {
			return 0, e
		}
		value := rng.Uint64()
		if value >= threshold {
			return int(value % n), nil
		}
	}
}

// Next selects from current progress; it does not mutate progress or reserve a
// case. Record the attempt before selecting again. Balance is by attempt count,
// learning filters out known cases, slowest uses mean successful unhinted time,
// and sequential follows the most recently recorded case in collection order.
func (s *Store) Next(ctx context.Context, c Collection, policy Policy, rng random.Source) (Case, error) {
	progress, e := s.Progress(ctx, c)
	if e != nil {
		return Case{}, e
	}
	if policy == "" {
		policy = Balanced
	}
	if policy == Sequential {
		latest := time.Time{}
		next := 0
		for i, p := range progress {
			for _, a := range p.Attempts {
				if a.At.After(latest) {
					latest = a.At
					next = (i + 1) % len(c.Cases)
				}
			}
		}
		return c.Cases[next], nil
	}
	if policy != Balanced && policy != LearningFirst && policy != Slowest {
		return Case{}, fmt.Errorf("policy must be balanced, learning, slowest or sequential")
	}
	candidates := []int{}
	best := float64(0)
	for i, p := range progress {
		if policy == LearningFirst && p.Status == Known {
			continue
		}
		score := -float64(len(p.Attempts))
		if policy == Slowest {
			var total int64
			count := 0
			for _, a := range p.Attempts {
				if a.Success && !a.Hint {
					total += a.DurationNS
					count++
				}
			}
			if count == 0 {
				score = 1 << 62
			} else {
				score = float64(total) / float64(count)
			}
		}
		if len(candidates) == 0 || score > best {
			best = score
			candidates = []int{i}
		} else if score == best {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return Case{}, fmt.Errorf("no cases match the selection policy")
	}
	index, e := bounded(ctx, rng, uint64(len(candidates)))
	if e != nil {
		return Case{}, e
	}
	return c.Cases[candidates[index]], nil
}
