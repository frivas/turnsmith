// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Adapted to Go for Turnsmith, 2026, from TNoodle threephase at
// d01a947d9f028f38085cda9b0507a9cf3d3f38a8 (cs.threephase). See COPYING for the GNU GPL terms.

package tnoodle444

import (
	"context"
	"github.com/frivas/turnsmith/internal/random"
	"strings"
)

var fullCubeCenterFacelet = [24]int{u5, u6, ua, u9, d5, d6, da, d9, f5, f6, fa, f9, b5, b6, ba, b9, r5, r6, ra, r9, l5, l6, la, l9}
var fullCubeEdgeFacelet = [24][2]int{
	{ud, f1}, {u4, l1}, {u2, b1}, {ub, r1}, {dd, be}, {d4, le}, {d2, fe}, {db, re}, {lb, f8}, {l4, b7}, {rb, b8}, {r4, f7},
	{f2, ue}, {l2, u8}, {b2, u1}, {r2, u7}, {bd, de}, {ld, d8}, {fd, d1}, {rd, d7}, {f4, l7}, {bb, l8}, {b4, r7}, {fb, r8}}
var fullCubeCornerFacelet = [8][3]int{{uf, r0, f3}, {uc, f0, l3}, {u0, l0, b3}, {u3, b0, r3},
	{d3, ff, rc}, {d0, lf, fc}, {dc, bf, lc}, {df, rf, bc}}

type FullCube struct {
	edge                                            *EdgeCube
	center                                          *CenterCube
	corner                                          *CornerCube
	value                                           int
	add1                                            bool
	length1, length2, length3                       int
	sym                                             int
	moveBuffer                                      [128]int
	moveLength, edgeAvail, centerAvail, cornerAvail int
}

func newFullCube() *FullCube {
	return &FullCube{edge: newEdgeCube(), center: newCenterCube(), corner: newCornerCube()}
}
func newFullCubeCopy(other *FullCube) *FullCube { c := newFullCube(); c.copy(other); return c }

// Random centers are a shuffled multiset, wings a full permutation, and corners
// an independent permutation with seven independent twists and a constrained
// eighth. Indistinguishable center labels absorb center/corner parity, as in the
// original 4x4 representation; wing permutation parity is unrestricted.
func newFullCubeRandom(r random.Source) *FullCube {
	return &FullCube{edge: newEdgeCubeRandom(r), center: newCenterCubeRandom(r), corner: newCornerCubeRandom(r)}
}
func newFullCubeRandomContext(ctx context.Context, r random.Source) (*FullCube, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	c := newFullCubeRandom(fullCubeContextSource{ctx: ctx, source: r})
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	return c, nil
}
func newFullCubeMoves(moves []int) *FullCube {
	c := newFullCube()
	for _, m := range moves {
		c.doMove(m)
	}
	return c
}
func newFullCubeFacelets(f []int) *FullCube {
	c := newFullCube()
	for i := 0; i < 24; i++ {
		c.center.ct[i] = f[fullCubeCenterFacelet[i]]
	}
	for i := 0; i < 24; i++ {
		for j := 0; j < 24; j++ {
			if f[fullCubeEdgeFacelet[i][0]] == fullCubeEdgeFacelet[j][0]/16 && f[fullCubeEdgeFacelet[i][1]] == fullCubeEdgeFacelet[j][1]/16 {
				c.edge.ep[i] = j
			}
		}
	}
	for i := 0; i < 8; i++ {
		ori := 0
		for ; ori < 3; ori++ {
			v := f[fullCubeCornerFacelet[i][ori]]
			if v == U || v == D {
				break
			}
		}
		col1 := f[fullCubeCornerFacelet[i][(ori+1)%3]]
		col2 := f[fullCubeCornerFacelet[i][(ori+2)%3]]
		for j := 0; j < 8; j++ {
			if col1 == fullCubeCornerFacelet[j][1]/16 && col2 == fullCubeCornerFacelet[j][2]/16 {
				c.corner.cp[i] = j
				c.corner.co[i] = ori % 3
				break
			}
		}
	}
	return c
}
func (c *FullCube) toFacelet(f []int) {
	for i := 0; i < 24; i++ {
		f[fullCubeCenterFacelet[i]] = c.center.ct[i]
	}
	for i := 0; i < 24; i++ {
		f[fullCubeEdgeFacelet[i][0]] = fullCubeEdgeFacelet[c.edge.ep[i]][0] / 16
		f[fullCubeEdgeFacelet[i][1]] = fullCubeEdgeFacelet[c.edge.ep[i]][1] / 16
	}
	for corn := 0; corn < 8; corn++ {
		j, ori := c.corner.cp[corn], c.corner.co[corn]
		for n := 0; n < 3; n++ {
			f[fullCubeCornerFacelet[corn][(n+ori)%3]] = fullCubeCornerFacelet[j][n] / 16
		}
	}
}
func (c *FullCube) facelets() [96]int {
	c.getEdge()
	c.getCenter()
	c.getCorner()
	var ret [96]int
	c.toFacelet(ret[:])
	return ret
}
func (c *FullCube) toString() string {
	f := c.facelets()
	var b strings.Builder
	for i, v := range f {
		b.WriteByte("URFDLB"[v])
		if i%4 == 3 {
			b.WriteByte('\n')
		}
		if i%16 == 15 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
func (c *FullCube) copy(other *FullCube) {
	c.edge.copy(other.edge)
	c.center.copy(other.center)
	c.corner.copy(other.corner)
	c.value = other.value
	c.add1 = other.add1
	c.length1 = other.length1
	c.length2 = other.length2
	c.length3 = other.length3
	c.sym = other.sym
	c.moveBuffer = other.moveBuffer
	c.moveLength = other.moveLength
	c.edgeAvail = other.edgeAvail
	c.centerAvail = other.centerAvail
	c.cornerAvail = other.cornerAvail
}
func (c *FullCube) checkEdge() bool { return c.getEdge().checkEdge() }

var fullCubeMove2rot = [9]int{35, 1, 34, 2, 4, 6, 22, 5, 19}

func (c *FullCube) getMoveString(inverse, rotation bool) string {
	offset := 0
	if c.add1 {
		offset = 2
	}
	fixedMoves := make([]int, c.moveLength-offset)
	idx := 0
	for i := 0; i < c.length1; i++ {
		fixedMoves[idx] = c.moveBuffer[i]
		idx++
	}
	sym := c.sym
	for i := c.length1 + offset; i < c.moveLength; i++ {
		m := center1Symmove[sym][c.moveBuffer[i]]
		if m >= dx1 {
			fixedMoves[idx] = m - 9
			sym = center1Symmult[sym][fullCubeMove2rot[m-dx1]]
		} else {
			fixedMoves[idx] = m
		}
		idx++
	}
	finishSym := center1Symmult[center1Syminv[sym]][center1GetSolvedSym(c.getCenter())]
	var b strings.Builder
	sym = finishSym
	if inverse {
		for i := idx - 1; i >= 0; i-- {
			m := fixedMoves[i]
			m = m/3*3 + 2 - m%3
			m = center1Symmove[sym][m]
			if m >= dx1 {
				b.WriteString(move2str[m-9])
				b.WriteByte(' ')
				sym = center1Symmult[sym][fullCubeMove2rot[m-dx1]]
			} else {
				b.WriteString(move2str[m])
				b.WriteByte(' ')
			}
		}
		if rotation {
			b.WriteString(center1Rot2str[center1Syminv[sym]])
			b.WriteByte(' ')
		}
	} else {
		for i := 0; i < idx; i++ {
			b.WriteString(move2str[fixedMoves[i]])
			b.WriteByte(' ')
		}
		if rotation {
			b.WriteString(center1Rot2str[finishSym])
		}
	}
	return b.String()
}
func (c *FullCube) to333Facelet() string {
	ret := make([]byte, 54)
	c.getEdge().fill333Facelet(ret)
	c.getCenter().fill333Facelet(ret)
	c.getCorner().fill333Facelet(ret)
	return string(ret)
}
func (c *FullCube) move(m int) {
	if c.moveLength >= len(c.moveBuffer) {
		panic("tnoodle444: move buffer capacity exceeded")
	}
	c.moveBuffer[c.moveLength] = m
	c.moveLength++
}
func (c *FullCube) doMove(m int) {
	c.getEdge().move(m)
	c.getCenter().move(m)
	c.getCorner().move(m % 18)
}
func (c *FullCube) getEdge() *EdgeCube {
	for c.edgeAvail < c.moveLength {
		c.edge.move(c.moveBuffer[c.edgeAvail])
		c.edgeAvail++
	}
	return c.edge
}
func (c *FullCube) getCenter() *CenterCube {
	for c.centerAvail < c.moveLength {
		c.center.move(c.moveBuffer[c.centerAvail])
		c.centerAvail++
	}
	return c.center
}
func (c *FullCube) getCorner() *CornerCube {
	for c.cornerAvail < c.moveLength {
		c.corner.move(c.moveBuffer[c.cornerAvail] % 18)
		c.cornerAvail++
	}
	return c.corner
}

// A canceled rejection sample must still leave random.Below. MaxUint64 passes
// every positive rejection threshold; the constructor's post-check discards it.
type fullCubeContextSource struct {
	ctx    context.Context
	source random.Source
}

func (s fullCubeContextSource) Uint64() uint64 {
	if s.ctx.Err() != nil {
		return ^uint64(0)
	}
	return s.source.Uint64()
}
