// Package alg parses bounded cube algorithms independently of generation.
package alg

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Move struct {
	Face  byte
	Width int
	Turns int
}

func (m Move) String() string {
	s := string(m.Face)
	if m.Width > 1 {
		if m.Width > 2 {
			s = strconv.Itoa(m.Width) + s
		}
		s += "w"
	}
	if m.Turns == 2 {
		s += "2"
	} else if m.Turns == 3 {
		s += "'"
	}
	return s
}
func (m Move) Inverse() Move { m.Turns = 4 - m.Turns; return m }
func Inverse(moves []Move) []Move {
	result := make([]Move, len(moves))
	for i, m := range moves {
		result[len(moves)-1-i] = m.Inverse()
	}
	return result
}
func Format(moves []Move) string {
	tokens := make([]string, len(moves))
	for i, m := range moves {
		tokens[i] = m.String()
	}
	return strings.Join(tokens, " ")
}
func Tokens(moves []Move) []string {
	tokens := make([]string, len(moves))
	for i, m := range moves {
		tokens[i] = m.String()
	}
	return tokens
}

type parser struct {
	text       []rune
	pos, limit int
}

func (p *parser) error(format string, args ...any) error {
	return fmt.Errorf("character %d: %s", p.pos+1, fmt.Sprintf(format, args...))
}
func (p *parser) space() {
	for p.pos < len(p.text) {
		if unicode.IsSpace(p.text[p.pos]) {
			p.pos++
			continue
		}
		if p.text[p.pos] == '#' || (p.text[p.pos] == '/' && p.pos+1 < len(p.text) && p.text[p.pos+1] == '/') {
			for p.pos < len(p.text) && p.text[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		break
	}
}
func (p *parser) append(dst, src []Move) ([]Move, error) {
	if len(src) > p.limit-len(dst) {
		return nil, p.error("expanded algorithm exceeds %d moves", p.limit)
	}
	return append(dst, src...), nil
}
func (p *parser) exponent(moves []Move) ([]Move, error) {
	start := p.pos
	for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
		p.pos++
	}
	n := 1
	if p.pos > start {
		value, e := strconv.ParseUint(string(p.text[start:p.pos]), 10, 32)
		if e != nil || value < 1 || value > uint64(p.limit) {
			return nil, p.error("repeat must be between 1 and %d", p.limit)
		}
		n = int(value)
	}
	if len(moves) > p.limit/n {
		return nil, p.error("expanded algorithm exceeds %d moves", p.limit)
	}
	out := make([]Move, 0, len(moves)*n)
	for i := 0; i < n; i++ {
		out = append(out, moves...)
	}
	if p.pos < len(p.text) && p.text[p.pos] == '\'' {
		out = Inverse(out)
		p.pos++
	}
	return out, nil
}
func (p *parser) sequence(stops string, depth int) ([]Move, error) {
	if depth > 32 {
		return nil, p.error("group nesting exceeds 32")
	}
	var out []Move
	for {
		p.space()
		if p.pos == len(p.text) || strings.ContainsRune(stops, p.text[p.pos]) {
			return out, nil
		}
		var part []Move
		var err error
		c := p.text[p.pos]
		switch c {
		case '(':
			p.pos++
			part, err = p.sequence(")", depth+1)
			if err != nil {
				return nil, err
			}
			if p.pos == len(p.text) || p.text[p.pos] != ')' {
				return nil, p.error("missing )")
			}
			p.pos++
			if len(part) == 0 {
				return nil, p.error("empty group")
			}
			part, err = p.exponent(part)
		case '[':
			p.pos++
			a, e := p.sequence(",:]", depth+1)
			if e != nil {
				return nil, e
			}
			if p.pos == len(p.text) || (p.text[p.pos] != ',' && p.text[p.pos] != ':') {
				return nil, p.error("expected , or : in bracket expression")
			}
			sep := p.text[p.pos]
			p.pos++
			b, e := p.sequence("]", depth+1)
			if e != nil {
				return nil, e
			}
			if p.pos == len(p.text) || p.text[p.pos] != ']' {
				return nil, p.error("missing ]")
			}
			p.pos++
			if len(a) == 0 || len(b) == 0 {
				return nil, p.error("empty bracket expression")
			}
			part, err = p.append(part, a)
			if err == nil {
				part, err = p.append(part, b)
			}
			if err == nil {
				part, err = p.append(part, Inverse(a))
			}
			if err == nil && sep == ',' {
				part, err = p.append(part, Inverse(b))
			}
			if err == nil {
				part, err = p.exponent(part)
			}
		default:
			width := 1
			if c >= '0' && c <= '9' {
				start := p.pos
				for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
					p.pos++
				}
				v, e := strconv.Atoi(string(p.text[start:p.pos]))
				if e != nil || v < 2 || v > 7 {
					return nil, p.error("wide move width must be between 2 and 7")
				}
				width = v
				if p.pos == len(p.text) {
					return nil, p.error("missing face after width")
				}
				c = p.text[p.pos]
			}
			if !strings.ContainsRune("UDLRFBMESxyzudlrfb", c) {
				return nil, p.error("unexpected token %q", c)
			}
			p.pos++
			lower := strings.ContainsRune("udlrfb", c)
			if lower {
				if width != 1 {
					return nil, p.error("numeric width requires an uppercase face with w suffix")
				}
				c = unicode.ToUpper(c)
				width = 2
			}
			wide := p.pos < len(p.text) && p.text[p.pos] == 'w'
			if wide {
				if !strings.ContainsRune("UDLRFB", c) {
					return nil, p.error("wide suffix requires an outer face")
				}
				p.pos++
				if width == 1 {
					width = 2
				}
			}
			if width > 1 && !lower && !wide {
				return nil, p.error("numeric width requires a w suffix")
			}
			move := Move{Face: byte(c), Width: width, Turns: 1}
			if p.pos < len(p.text) && p.text[p.pos] == '2' {
				move.Turns = 2
				p.pos++
			}
			if p.pos < len(p.text) && p.text[p.pos] == '\'' {
				if move.Turns != 2 {
					move.Turns = 3
				}
				p.pos++
			}
			part = []Move{move}
		}
		if err != nil {
			return nil, err
		}
		out, err = p.append(out, part)
		if err != nil {
			return nil, err
		}
	}
}
func Parse(text string, limit int) ([]Move, error) {
	if len(text) > 1<<20 {
		return nil, fmt.Errorf("algorithm input exceeds 1 MiB")
	}
	if limit < 1 {
		return nil, fmt.Errorf("move limit must be positive")
	}
	text = strings.NewReplacer("’", "'", "′", "'").Replace(text)
	p := parser{text: []rune(text), limit: limit}
	moves, err := p.sequence("", 0)
	if err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return nil, fmt.Errorf("algorithm contains no moves")
	}
	return moves, nil
}
