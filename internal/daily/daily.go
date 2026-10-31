// Package daily constructs portable practice sets with a public UTC seed policy.
package daily

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"github.com/frivas/turnsmith/internal/app"
	"time"
)

const SeedPolicyVersion = "1"

type Options struct {
	Date   string
	Name   string
	Puzzle string
	Method string
	Count  int
	Length *int
}
type Manifest struct {
	SchemaVersion     int        `json:"schema_version"`
	Kind              string     `json:"kind"`
	Name              string     `json:"name"`
	Date              string     `json:"date"`
	DatePolicy        string     `json:"date_policy"`
	SeedPolicyVersion string     `json:"seed_policy_version"`
	Result            app.Result `json:"result"`
}

// UTCDate returns the UTC calendar date of an instant, regardless of its location.
func UTCDate(now time.Time) string { return now.UTC().Format(time.DateOnly) }

func validateDate(date string) error {
	parsed, err := time.Parse(time.DateOnly, date)
	if err != nil || len(date) != 10 || parsed.Format(time.DateOnly) != date {
		return app.Invalid("date must be a valid YYYY-MM-DD calendar date")
	}
	return nil
}

// DeriveSeed hashes length-prefixed UTF-8 fields in this order: domain,
// seed-policy version, UTC calendar date, canonical puzzle ID, method ID.
// The first eight hash bytes form an unsigned big-endian root seed.
// Names, counts and lengths do not change the seed. Generator algorithm
// versions remain recorded in Result and participate in app stream derivation.
func DeriveSeed(date, puzzle, method string) (uint64, error) {
	if err := validateDate(date); err != nil {
		return 0, err
	}
	if puzzle == "" || method == "" {
		return 0, app.Invalid("daily seed requires canonical puzzle and method IDs")
	}
	h := sha256.New()
	for _, field := range []string{"turnsmith/daily", SeedPolicyVersion, date, puzzle, method} {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(field)))
		h.Write(size[:])
		h.Write([]byte(field))
	}
	return binary.BigEndian.Uint64(h.Sum(nil)[:8]), nil
}

func Build(ctx context.Context, service app.Service, options Options) (Manifest, error) {
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	date := options.Date
	if date == "" {
		date = UTCDate(time.Now())
	}
	if err := validateDate(date); err != nil {
		return Manifest{}, err
	}
	id := options.Puzzle
	if id == "" {
		id = "333"
	}
	descriptor, ok := service.Registry.Resolve(id)
	if !ok {
		return Manifest{}, app.Invalid("unsupported puzzle %q", id)
	}
	method := options.Method
	if method == "" {
		method = descriptor.DefaultMethod
	}
	if _, ok := service.Registry.Generator(descriptor.ID, method); !ok {
		return Manifest{}, app.Invalid("puzzle %q does not support method %q", descriptor.ID, method)
	}
	seed, err := DeriveSeed(date, descriptor.ID, method)
	if err != nil {
		return Manifest{}, err
	}
	result, err := service.Generate(ctx, app.Request{Puzzle: descriptor.ID, Method: method, Count: options.Count, Length: options.Length, Seed: &seed})
	if err != nil {
		return Manifest{}, err
	}
	result.Randomness = "seeded-practice"
	name := options.Name
	if name == "" {
		name = "Daily practice " + date
	}
	return Manifest{SchemaVersion: 1, Kind: "practice-set", Name: name, Date: date, DatePolicy: "UTC", SeedPolicyVersion: SeedPolicyVersion, Result: result}, nil
}
