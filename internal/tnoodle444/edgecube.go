// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Adapted to Go for Turnsmith, 2026, from TNoodle threephase at
// d01a947d9f028f38085cda9b0507a9cf3d3f38a8 (cs.threephase). See COPYING for the GNU GPL terms.

package tnoodle444

import "github.com/frivas/turnsmith/internal/random"

type EdgeCube struct{ ep [24]int }

var edgeCubeEdgeColor = [12][2]int{{F, U}, {L, U}, {B, U}, {R, U}, {B, D}, {L, D}, {F, D}, {R, D}, {F, L}, {B, L}, {B, R}, {F, R}}
var edgeCubeEdgeMap = [24]int{F2, L2, B2, R2, B8, L8, F8, R8, F4, B6, B4, F6, U8, U4, U2, U6, D8, D4, D2, D6, L6, L4, R6, R4}

func newEdgeCube() *EdgeCube {
	c := &EdgeCube{}
	for i := range c.ep {
		c.ep[i] = i
	}
	return c
}
func (c *EdgeCube) getParity() int { return parity(c.ep[:]) }
func (c *EdgeCube) fill333Facelet(facelet []byte) {
	for i := 0; i < 24; i++ {
		facelet[edgeCubeEdgeMap[i]] = "URFDLB"[edgeCubeEdgeColor[c.ep[i]%12][c.ep[i]/12]]
	}
}
func (c *EdgeCube) checkEdge() bool {
	ck := 0
	p := false
	for i := 0; i < 12; i++ {
		ck |= 1 << c.ep[i]
		p = p != (c.ep[i] >= 12)
	}
	ck &= ck >> 12
	return ck == 0 && !p
}
func newEdgeCubeCopy(other *EdgeCube) *EdgeCube { c := &EdgeCube{}; c.copy(other); return c }
func (c *EdgeCube) copy(other *EdgeCube)        { c.ep = other.ep }
func newEdgeCubeRandom(r random.Source) *EdgeCube {
	c := newEdgeCube()
	for i := 0; i < 23; i++ {
		t := i + int(random.Below(r, uint64(24-i)))
		c.ep[i], c.ep[t] = c.ep[t], c.ep[i]
	}
	return c
}
func (c *EdgeCube) move(m int) {
	key := m % 3
	m /= 3
	switch m {
	case 0: //U
		swap(c.ep[:], 0, 1, 2, 3, key)
		swap(c.ep[:], 12, 13, 14, 15, key)

	case 1: //R
		swap(c.ep[:], 11, 15, 10, 19, key)
		swap(c.ep[:], 23, 3, 22, 7, key)

	case 2: //F
		swap(c.ep[:], 0, 11, 6, 8, key)
		swap(c.ep[:], 12, 23, 18, 20, key)

	case 3: //D
		swap(c.ep[:], 4, 5, 6, 7, key)
		swap(c.ep[:], 16, 17, 18, 19, key)

	case 4: //L
		swap(c.ep[:], 1, 20, 5, 21, key)
		swap(c.ep[:], 13, 8, 17, 9, key)

	case 5: //B
		swap(c.ep[:], 2, 9, 4, 10, key)
		swap(c.ep[:], 14, 21, 16, 22, key)

	case 6: //u
		swap(c.ep[:], 0, 1, 2, 3, key)
		swap(c.ep[:], 12, 13, 14, 15, key)
		swap(c.ep[:], 9, 22, 11, 20, key)

	case 7: //r
		swap(c.ep[:], 11, 15, 10, 19, key)
		swap(c.ep[:], 23, 3, 22, 7, key)
		swap(c.ep[:], 2, 16, 6, 12, key)

	case 8: //f
		swap(c.ep[:], 0, 11, 6, 8, key)
		swap(c.ep[:], 12, 23, 18, 20, key)
		swap(c.ep[:], 3, 19, 5, 13, key)

	case 9: //d
		swap(c.ep[:], 4, 5, 6, 7, key)
		swap(c.ep[:], 16, 17, 18, 19, key)
		swap(c.ep[:], 8, 23, 10, 21, key)

	case 10: //l
		swap(c.ep[:], 1, 20, 5, 21, key)
		swap(c.ep[:], 13, 8, 17, 9, key)
		swap(c.ep[:], 14, 0, 18, 4, key)

	case 11: //b
		swap(c.ep[:], 2, 9, 4, 10, key)
		swap(c.ep[:], 14, 21, 16, 22, key)
		swap(c.ep[:], 7, 15, 1, 17, key)

	}
}
