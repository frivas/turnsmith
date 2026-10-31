// Package puzzle describes capabilities independently of CLI and notation.
package puzzle

import (
	"context"
	"fmt"
	"sort"

	"github.com/frivas/turnsmith/internal/random"
)

type GeneratorDescriptor struct {
	ID               string `json:"id"`
	AlgorithmVersion string `json:"algorithm_version"`
	Description      string `json:"description"`
	ExplicitLength   bool   `json:"explicit_length"`
	DirectEntropy    bool   `json:"direct_entropy,omitempty"`
	DefaultLength    *int   `json:"default_length,omitempty"`
	MinLength        int    `json:"min_length,omitempty"`
	MaxLength        int    `json:"max_length,omitempty"`
	Guarantees       string `json:"guarantees"`
	Limitations      string `json:"limitations"`
}
type Descriptor struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	Aliases       []string              `json:"aliases"`
	DefaultMethod string                `json:"default_method"`
	Methods       []GeneratorDescriptor `json:"methods"`
}
type Options struct{ Length *int }
type Generated struct{ Moves []string }
type Generator interface {
	Descriptor() GeneratorDescriptor
	Generate(context.Context, random.Source, Options) (Generated, error)
}
type Registration struct {
	Puzzle     Descriptor
	Generators []Generator
}
type entry struct {
	descriptor Descriptor
	generators map[string]Generator
}
type Registry struct {
	entries map[string]entry
	aliases map[string]string
}

func New(registrations ...Registration) (*Registry, error) {
	r := &Registry{entries: make(map[string]entry), aliases: make(map[string]string)}
	for _, reg := range registrations {
		d := reg.Puzzle
		if d.ID == "" || d.Name == "" {
			return nil, fmt.Errorf("puzzle ID and name are required")
		}
		names := append([]string{d.ID}, d.Aliases...)
		for _, name := range names {
			if name == "" {
				return nil, fmt.Errorf("empty puzzle alias")
			}
			if _, exists := r.aliases[name]; exists {
				return nil, fmt.Errorf("duplicate puzzle ID or alias %q", name)
			}
			r.aliases[name] = d.ID
		}
		e := entry{generators: make(map[string]Generator)}
		d.Aliases = append([]string{}, d.Aliases...)
		d.Methods = []GeneratorDescriptor{}
		for _, g := range reg.Generators {
			m := cloneMethod(g.Descriptor())
			if m.ID == "" || m.AlgorithmVersion == "" {
				return nil, fmt.Errorf("method ID and algorithm version are required")
			}
			if _, exists := e.generators[m.ID]; exists {
				return nil, fmt.Errorf("duplicate method %q", m.ID)
			}
			if m.ExplicitLength && (m.DefaultLength == nil || m.MinLength < 1 || m.MaxLength < m.MinLength || *m.DefaultLength < m.MinLength || *m.DefaultLength > m.MaxLength) {
				return nil, fmt.Errorf("invalid length capabilities for %q", m.ID)
			}
			if !m.ExplicitLength && m.DefaultLength != nil {
				return nil, fmt.Errorf("method %q declares an inapplicable default length", m.ID)
			}
			e.generators[m.ID] = g
			d.Methods = append(d.Methods, m)
		}
		if _, ok := e.generators[d.DefaultMethod]; !ok {
			return nil, fmt.Errorf("missing default method for puzzle %q", d.ID)
		}
		sort.Slice(d.Methods, func(i, j int) bool { return d.Methods[i].ID < d.Methods[j].ID })
		e.descriptor = d
		r.entries[d.ID] = e
	}
	return r, nil
}
func cloneMethod(m GeneratorDescriptor) GeneratorDescriptor {
	if m.DefaultLength != nil {
		n := *m.DefaultLength
		m.DefaultLength = &n
	}
	return m
}
func clone(d Descriptor) Descriptor {
	d.Aliases = append([]string{}, d.Aliases...)
	d.Methods = append([]GeneratorDescriptor{}, d.Methods...)
	for i := range d.Methods {
		d.Methods[i] = cloneMethod(d.Methods[i])
	}
	return d
}
func (r *Registry) Resolve(id string) (Descriptor, bool) {
	canonical, ok := r.aliases[id]
	if !ok {
		return Descriptor{}, false
	}
	return clone(r.entries[canonical].descriptor), true
}
func (r *Registry) Generator(id, method string) (Generator, bool) {
	e, ok := r.entries[id]
	if !ok {
		return nil, false
	}
	g, ok := e.generators[method]
	return g, ok
}
func (r *Registry) List() []Descriptor {
	result := make([]Descriptor, 0, len(r.entries))
	for _, e := range r.entries {
		result = append(result, clone(e.descriptor))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
