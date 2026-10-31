package training

import (
	"fmt"
	"time"
)

type Status string

const (
	New      Status = "new"
	Learning Status = "learning"
	Known    Status = "known"
)

type Policy string

const (
	Balanced      Policy = "balanced"
	LearningFirst Policy = "learning"
	Slowest       Policy = "slowest"
	Sequential    Policy = "sequential"
)

type Attempt struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	DurationNS int64     `json:"duration_ns"`
	Hint       bool      `json:"hint"`
	Success    bool      `json:"success"`
}
type CaseProgress struct {
	CaseID   string    `json:"case_id"`
	Status   Status    `json:"status"`
	Attempts []Attempt `json:"attempts"`
}
type CollectionProgress struct {
	CollectionID string         `json:"collection_id"`
	Name         string         `json:"name"`
	Cases        []CaseProgress `json:"cases"`
}
type Document struct {
	SchemaVersion int                  `json:"schema_version"`
	Collections   []CollectionProgress `json:"collections"`
}

func validStatus(s Status) bool { return s == New || s == Learning || s == Known }
func validate(doc Document) error {
	if doc.SchemaVersion != 1 {
		return fmt.Errorf("unsupported training schema version %d", doc.SchemaVersion)
	}
	if len(doc.Collections) > 10000 {
		return fmt.Errorf("too many collections")
	}
	ids := map[string]bool{}
	total := 0
	for _, c := range doc.Collections {
		if len(c.CollectionID) != 64 || ids[c.CollectionID] || !textOK(c.Name, 200) {
			return fmt.Errorf("invalid collection metadata")
		}
		ids[c.CollectionID] = true
		cases := map[string]bool{}
		for _, p := range c.Cases {
			if !textOK(p.CaseID, 200) || cases[p.CaseID] || !validStatus(p.Status) {
				return fmt.Errorf("invalid case progress")
			}
			cases[p.CaseID] = true
			attempts := map[string]bool{}
			for _, a := range p.Attempts {
				total++
				if !textOK(a.ID, 200) || attempts[a.ID] || a.At.IsZero() || a.DurationNS < 0 || a.DurationNS > int64(24*time.Hour) {
					return fmt.Errorf("invalid training attempt")
				}
				attempts[a.ID] = true
			}
		}
	}
	if total > 100000 {
		return fmt.Errorf("training progress exceeds 100000 attempts")
	}
	return nil
}
