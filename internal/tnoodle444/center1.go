// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen
// Go adaptation Copyright (C) 2026 Francisco Rivas and contributors.
// Derived from TNoodle-lib cs.threephase, commit d01a947d9f028f38085cda9b0507a9cf3d3f38a8.
// This program is free software under GNU GPL version 3 or later, without any
// warranty. See https://www.gnu.org/licenses/gpl-3.0.html for the full license.

package tnoodle444

import (
	"context"
	"fmt"
	"sort"
)

var (
	center1Ctsmv   [15582][36]int
	center1Sym2raw [15582]int
	center1Csprun  [15582]int
	center1Symmult [48][48]int
	center1Symmove [48][36]int
	center1Syminv  [48]int
	center1Finish  [48]int
	center1Raw2sym []int
	center1Rot2str = [48]string{"", "y2", "x", "x y2", "x2", "z2", "x'", "x' y2", "", "", "", "", "", "", "", "", "y z", "y' z'", "y2 z", "z'", "y' z", "y z'", "z", "z y2", "", "", "", "", "", "", "", "", "y' x'", "y x", "y'", "y", "y' x", "y x'", "y z2", "y' z2", "", "", "", "", "", "", "", ""}
)

type Center1 struct{ ct [24]int }

func newCenter1() *Center1 {
	c := &Center1{}
	for i := 0; i < 8; i++ {
		c.ct[i] = 1
	}
	return c
}
func newCenter1FromValues(values []int) *Center1 { c := &Center1{}; copy(c.ct[:], values); return c }
func newCenter1FromCube(cube *CenterCube, urf int) *Center1 {
	c := &Center1{}
	for i := range c.ct {
		if cube.ct[i]%3 == urf {
			c.ct[i] = 1
		}
	}
	return c
}
func (c *Center1) copy(other *Center1) { c.ct = other.ct }
func (c *Center1) set(idx int) {
	r := 8
	for i := 23; i >= 0; i-- {
		c.ct[i] = 0
		if idx >= Cnk[i][r] {
			idx -= Cnk[i][r]
			r--
			c.ct[i] = 1
		}
	}
}
func (c *Center1) get() int {
	idx, r := 0, 8
	for i := 23; i >= 0; i-- {
		if c.ct[i] == 1 {
			idx += Cnk[i][r]
			r--
		}
	}
	return idx
}
func center1Raw2symIndex(n int) int {
	i := sort.SearchInts(center1Sym2raw[:], n)
	if i < len(center1Sym2raw) && center1Sym2raw[i] == n {
		return i
	}
	return -1
}
func (c *Center1) getsym() int {
	if center1Raw2sym != nil {
		return center1Raw2sym[c.get()]
	}
	for j := 0; j < 48; j++ {
		if coordinate := center1Raw2symIndex(c.get()); coordinate != -1 {
			return coordinate*64 + j
		}
		c.nextSymmetry(j)
	}
	return -1
}
func (c *Center1) nextSymmetry(j int) {
	c.rot(0)
	if j%2 == 1 {
		c.rot(1)
	}
	if j%8 == 7 {
		c.rot(2)
	}
	if j%16 == 15 {
		c.rot(3)
	}
}
func (c *Center1) rotate(r int) {
	for j := 0; j < r; j++ {
		c.nextSymmetry(j)
	}
}
func (c *Center1) rot(r int) {
	switch r {
	case 0:
		c.move(ux2)
		c.move(dx2)
	case 1:
		c.move(rx1)
		c.move(lx3)
	case 2:
		swap(c.ct[:], 0, 3, 1, 2, 1)
		swap(c.ct[:], 8, 11, 9, 10, 1)
		swap(c.ct[:], 4, 7, 5, 6, 1)
		swap(c.ct[:], 12, 15, 13, 14, 1)
		swap(c.ct[:], 16, 19, 21, 22, 1)
		swap(c.ct[:], 17, 18, 20, 23, 1)
	case 3:
		c.move(ux1)
		c.move(dx3)
		c.move(fx1)
		c.move(bx3)
	}
}
func (c *Center1) move(m int) {
	key := m % 3
	switch m / 3 {
	case 0:
		swap(c.ct[:], 0, 1, 2, 3, key)
	case 1:
		swap(c.ct[:], 16, 17, 18, 19, key)
	case 2:
		swap(c.ct[:], 8, 9, 10, 11, key)
	case 3:
		swap(c.ct[:], 4, 5, 6, 7, key)
	case 4:
		swap(c.ct[:], 20, 21, 22, 23, key)
	case 5:
		swap(c.ct[:], 12, 13, 14, 15, key)
	case 6:
		swap(c.ct[:], 0, 1, 2, 3, key)
		swap(c.ct[:], 8, 20, 12, 16, key)
		swap(c.ct[:], 9, 21, 13, 17, key)
	case 7:
		swap(c.ct[:], 16, 17, 18, 19, key)
		swap(c.ct[:], 1, 15, 5, 9, key)
		swap(c.ct[:], 2, 12, 6, 10, key)
	case 8:
		swap(c.ct[:], 8, 9, 10, 11, key)
		swap(c.ct[:], 2, 19, 4, 21, key)
		swap(c.ct[:], 3, 16, 5, 22, key)
	case 9:
		swap(c.ct[:], 4, 5, 6, 7, key)
		swap(c.ct[:], 10, 18, 14, 22, key)
		swap(c.ct[:], 11, 19, 15, 23, key)
	case 10:
		swap(c.ct[:], 20, 21, 22, 23, key)
		swap(c.ct[:], 0, 8, 4, 14, key)
		swap(c.ct[:], 3, 11, 7, 13, key)
	case 11:
		swap(c.ct[:], 12, 13, 14, 15, key)
		swap(c.ct[:], 1, 20, 7, 18, key)
		swap(c.ct[:], 0, 23, 6, 17, key)
	}
}
func center1GetSolvedSym(cube *CenterCube) int {
	c := newCenter1FromValues(cube.ct[:])
	for j := 0; j < 48; j++ {
		solved := true
		for i := range c.ct {
			if c.ct[i] != fullCubeCenterFacelet[i]/16 {
				solved = false
				break
			}
		}
		if solved {
			return j
		}
		c.nextSymmetry(j)
	}
	return -1
}

func center1InitSym2Raw(ctx context.Context) error {
	c := newCenter1()
	occupied := make([]uint32, 735471/32+1)
	count := 0
	for i := 0; i < 735471; i++ {
		if i&1023 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if occupied[i>>5]&(uint32(1)<<uint(i&31)) != 0 {
			continue
		}
		c.set(i)
		for j := 0; j < 48; j++ {
			idx := c.get()
			occupied[idx>>5] |= uint32(1) << uint(idx&31)
			if center1Raw2sym != nil {
				center1Raw2sym[idx] = count<<6 | center1Syminv[j]
			}
			c.nextSymmetry(j)
		}
		if count >= len(center1Sym2raw) {
			return fmt.Errorf("phase1 symmetry representatives exceed %d", len(center1Sym2raw))
		}
		center1Sym2raw[count] = i
		count++
	}
	if count != len(center1Sym2raw) {
		return fmt.Errorf("phase1 symmetry representatives: got %d, want %d", count, len(center1Sym2raw))
	}
	return ctx.Err()
}
func center1CreateMoveTable(ctx context.Context) error {
	c, d := newCenter1(), newCenter1()
	for i := range center1Ctsmv {
		if i&255 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		d.set(center1Sym2raw[i])
		for m := 0; m < 36; m++ {
			c.copy(d)
			c.move(m)
			center1Ctsmv[i][m] = c.getsym()
		}
	}
	return ctx.Err()
}
func center1CreatePrun(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i := range center1Csprun {
		center1Csprun[i] = -1
	}
	center1Csprun[0] = 0
	depth, done := 0, 1
	for done != len(center1Csprun) {
		inverse := depth > 4
		selected, check := depth, -1
		if inverse {
			selected, check = -1, depth
		}
		depth++
		before := done
		for i := range center1Csprun {
			if i&1023 == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			if center1Csprun[i] != selected {
				continue
			}
			for m := 0; m < 27; m++ {
				idx := center1Ctsmv[i][m] >> 6
				if center1Csprun[idx] != check {
					continue
				}
				done++
				if inverse {
					center1Csprun[i] = depth
					break
				}
				center1Csprun[idx] = depth
			}
		}
		if done == before {
			return fmt.Errorf("phase1 pruning did not reach every coordinate")
		}
	}
	return ctx.Err()
}
func center1InitSym(ctx context.Context) error {
	c := newCenter1()
	for i := range c.ct {
		c.ct[i] = i
	}
	d, e, f := newCenter1FromValues(c.ct[:]), newCenter1FromValues(c.ct[:]), newCenter1FromValues(c.ct[:])
	for i := 0; i < 48; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for j := 0; j < 48; j++ {
			for k := 0; k < 48; k++ {
				if c.ct == d.ct {
					center1Symmult[i][j] = k
					if k == 0 {
						center1Syminv[i] = j
					}
				}
				d.nextSymmetry(k)
			}
			c.nextSymmetry(j)
		}
		c.nextSymmetry(i)
	}
	for i := 0; i < 48; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		c.copy(e)
		c.rotate(center1Syminv[i])
		for j := 0; j < 36; j++ {
			d.copy(c)
			d.move(j)
			d.rotate(i)
			for k := 0; k < 36; k++ {
				f.copy(e)
				f.move(k)
				if f.ct == d.ct {
					center1Symmove[i][j] = k
					break
				}
			}
		}
	}
	c.set(0)
	for i := 0; i < 48; i++ {
		center1Finish[center1Syminv[i]] = c.get()
		c.nextSymmetry(i)
	}
	return ctx.Err()
}
