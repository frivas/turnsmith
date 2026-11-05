// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Adapted to Go for Turnsmith, 2026, from TNoodle threephase at
// d01a947d9f028f38085cda9b0507a9cf3d3f38a8 (cs.threephase). See COPYING for the GNU GPL terms.
package tnoodle444

import "github.com/frivas/turnsmith/internal/random"

type CornerCube struct{ cp, co [8]int }

var cornerCubeMoveCube [18]*CornerCube
var cornerCubeCornerFacelet = [8][3]int{{U9, R1, F3}, {U7, F1, L3}, {U1, L1, B3}, {U3, B1, R3}, {D3, F9, R7}, {D1, L9, F7}, {D7, B9, L7}, {D9, R9, B7}}

func newCornerCube() *CornerCube {
	c := &CornerCube{}
	for i := range c.cp {
		c.cp[i] = i
	}
	return c
}
func newCornerCubeCoords(cperm, twist int) *CornerCube {
	c := newCornerCube()
	c.setCPerm(cperm)
	c.setTwist(twist)
	return c
}
func newCornerCubeRandom(r random.Source) *CornerCube {
	return newCornerCubeCoords(int(random.Below(r, 40320)), int(random.Below(r, 2187)))
}
func (c *CornerCube) copy(other *CornerCube) { c.cp, c.co = other.cp, other.co }
func (c *CornerCube) getParity() int         { return parity(c.cp[:]) }
func (c *CornerCube) fill333Facelet(facelet []byte) {
	for corn := 0; corn < 8; corn++ {
		j, ori := c.cp[corn], c.co[corn]
		for n := 0; n < 3; n++ {
			facelet[cornerCubeCornerFacelet[corn][(n+ori)%3]] = "URFDLB"[cornerCubeCornerFacelet[j][n]/9]
		}
	}
}
func cornerCubeCornMult(a, b, prod *CornerCube) {
	for corn := 0; corn < 8; corn++ {
		prod.cp[corn] = a.cp[b.cp[corn]]
		oriA, oriB := a.co[b.cp[corn]], b.co[corn]
		ori := oriA
		if oriA < 3 {
			ori += oriB
		} else {
			ori += 6 - oriB
		}
		ori %= 3
		if (oriA >= 3) != (oriB >= 3) {
			ori += 3
		}
		prod.co[corn] = ori
	}
}
func (c *CornerCube) setTwist(idx int) {
	twst := 0
	for i := 6; i >= 0; i-- {
		c.co[i] = idx % 3
		twst += c.co[i]
		idx /= 3
	}
	c.co[7] = (15 - twst) % 3
}
func (c *CornerCube) setCPerm(idx int) { set8Perm(c.cp[:], idx) }
func (c *CornerCube) move(idx int) {
	var out CornerCube
	cornerCubeCornMult(c, cornerCubeMoveCube[idx], &out)
	c.copy(&out)
}
func init() {
	// Do not use set8Perm here: the factorial table is initialized independently
	// of file order. Build it before constructing the immutable move cubes.
	fact[0] = 1
	for i := 0; i < 12; i++ {
		fact[i+1] = fact[i] * (i + 1)
	}
	for _, entry := range [][3]int{{0, 15120, 0}, {3, 21021, 1494}, {6, 8064, 1236}, {9, 9, 0}, {12, 1230, 412}, {15, 224, 137}} {
		cornerCubeMoveCube[entry[0]] = newCornerCubeCoords(entry[1], entry[2])
	}
	for a := 0; a < 18; a += 3 {
		for p := 0; p < 2; p++ {
			cornerCubeMoveCube[a+p+1] = newCornerCube()
			cornerCubeCornMult(cornerCubeMoveCube[a+p], cornerCubeMoveCube[a], cornerCubeMoveCube[a+p+1])
		}
	}
}
