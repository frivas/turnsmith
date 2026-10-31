package daily

import (
	"context"
	"errors"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
	"reflect"
	"testing"
	"time"
)

func service(t *testing.T) app.Service {
	t.Helper()
	r, e := puzzle.New(cube333.Registration())
	if e != nil {
		t.Fatal(e)
	}
	return app.Service{Registry: r, Version: "test"}
}
func TestSharedPrefix(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	a, e := Build(ctx, s, Options{Date: "2026-10-31", Count: 2})
	if e != nil {
		t.Fatal(e)
	}
	b, e := Build(ctx, s, Options{Date: "2026-10-31", Name: "Club night", Puzzle: "3x3x3", Method: "random-move", Count: 5})
	if e != nil {
		t.Fatal(e)
	}
	if a.Result.Seed != b.Result.Seed || !reflect.DeepEqual(a.Result.Scrambles, b.Result.Scrambles[:2]) {
		t.Fatal("batch prefix or alias changed stream")
	}
	c, e := Build(ctx, s, Options{Date: "2026-10-31", Count: 2})
	if e != nil || !reflect.DeepEqual(a, c) {
		t.Fatal("not repeatable", e)
	}
	if a.Kind != "practice-set" || a.DatePolicy != "UTC" || a.SchemaVersion != 1 || a.SeedPolicyVersion != "1" {
		t.Fatal("metadata", a)
	}
	if a.Result.Seed != "14884999549874360842" {
		t.Fatal("seed policy changed", a.Result.Seed)
	}
}
func TestDatePolicy(t *testing.T) {
	instant := time.Date(2026, 10, 31, 0, 30, 0, 0, time.FixedZone("east", 7200))
	if UTCDate(instant) != "2026-10-30" {
		t.Fatal("local date leaked")
	}
	if UTCDate(instant.In(time.FixedZone("west", -18000))) != UTCDate(instant) {
		t.Fatal("location changed UTC date")
	}
	for _, date := range []string{"2026-2-01", "2026-02-29", "2026-11-31", "2026-10-31T00:00:00Z", " 2026-10-31", "", "2026-10-31\n"} {
		_, e := DeriveSeed(date, "333", "random-move")
		var invalid *app.InvalidInput
		if !errors.As(e, &invalid) {
			t.Fatalf("accepted invalid date %q: %v", date, e)
		}
	}
	if _, e := DeriveSeed("2024-02-29", "333", "random-move"); e != nil {
		t.Fatal(e)
	}
}
func TestDailyDomains(t *testing.T) {
	a, _ := DeriveSeed("2026-10-31", "333", "random-move")
	for _, v := range [][3]string{{"2026-11-01", "333", "random-move"}, {"2026-10-31", "444", "random-move"}, {"2026-10-31", "333", "random-state"}} {
		b, e := DeriveSeed(v[0], v[1], v[2])
		if e != nil || a == b {
			t.Fatal("seed domain collision", v, e)
		}
	}
}
func TestDailyErrors(t *testing.T) {
	s := service(t)
	for _, o := range []Options{{Date: "bad", Count: 1}, {Date: "2026-10-31", Puzzle: "bad", Count: 1}, {Date: "2026-10-31", Method: "bad", Count: 1}, {Date: "2026-10-31", Count: 0}} {
		if _, e := Build(context.Background(), s, o); e == nil {
			t.Fatal("accepted", o)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Build(ctx, s, Options{Count: 1}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
