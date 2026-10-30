package puzzle_test

import (
	"reflect"
	"testing"

	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
)

func registry(t testing.TB) *puzzle.Registry {
	t.Helper()
	r, err := puzzle.New(cube333.Registration())
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestResolveAndIsolation(t *testing.T) {
	r := registry(t)
	for _, id := range []string{"333", "3x3", "3x3x3"} {
		d, ok := r.Resolve(id)
		if !ok || d.ID != "333" {
			t.Fatal(id, d, ok)
		}
	}
	for _, id := range []string{"444", "222", "pyraminx", "", " 333", "3X3"} {
		if _, ok := r.Resolve(id); ok {
			t.Fatalf("resolved unsupported %q", id)
		}
	}
	first := r.List()
	first[0].Aliases[0] = "broken"
	first[0].Methods[0].ID = "broken"
	*first[0].Methods[0].DefaultLength = 999
	d, _ := r.Resolve("333")
	if d.Aliases[0] != "3x3" || d.Methods[0].ID != "random-move" || *d.Methods[0].DefaultLength != 25 {
		t.Fatal("registry mutated", d)
	}
	if _, ok := r.Generator("333", "random-state"); ok {
		t.Fatal("unsupported method")
	}
	if _, ok := r.Generator("444", "random-move"); ok {
		t.Fatal("unsupported puzzle")
	}
}
func TestCollisions(t *testing.T) {
	a := cube333.Registration()
	for _, change := range []func(*puzzle.Registration){
		func(b *puzzle.Registration) {},
		func(b *puzzle.Registration) { b.Puzzle.ID = "444"; b.Puzzle.Aliases = []string{"3x3"} },
		func(b *puzzle.Registration) { b.Puzzle.ID = "3x3"; b.Puzzle.Aliases = nil },
	} {
		b := cube333.Registration()
		change(&b)
		if _, err := puzzle.New(a, b); err == nil {
			t.Fatal("collision accepted")
		}
	}
	for _, change := range []func(*puzzle.Registration){
		func(b *puzzle.Registration) { b.Puzzle.Aliases = []string{"same", "same"} },
		func(b *puzzle.Registration) { b.Puzzle.ID = "" },
		func(b *puzzle.Registration) { b.Puzzle.Aliases = []string{""} },
		func(b *puzzle.Registration) { b.Puzzle.DefaultMethod = "missing" },
		func(b *puzzle.Registration) { b.Generators = append(b.Generators, b.Generators[0]) },
	} {
		b := cube333.Registration()
		change(&b)
		if _, err := puzzle.New(b); err == nil {
			t.Fatal("invalid registration accepted")
		}
	}
}
func TestStableOrder(t *testing.T) {
	a := cube333.Registration()
	b := cube333.Registration()
	b.Puzzle.ID = "aaa"
	b.Puzzle.Aliases = nil
	r, err := puzzle.New(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{r.List()[0].ID, r.List()[1].ID}; !reflect.DeepEqual(got, []string{"333", "aaa"}) {
		t.Fatal(got)
	}
}
func FuzzResolution(f *testing.F) {
	for _, s := range []string{"333", "3x3", "3x3x3", "444", "", "pyraminx"} {
		f.Add(s)
	}
	r := registry(f)
	f.Fuzz(func(t *testing.T, s string) {
		d, ok := r.Resolve(s)
		want := s == "333" || s == "3x3" || s == "3x3x3"
		if ok != want || (ok && d.ID != "333") {
			t.Fatal(s, d, ok)
		}
	})
}
