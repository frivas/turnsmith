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
)

var (
	center2Rlmv   [70][28]int
	center2Ctmv   [6435][28]int
	center2Rlrot  [70][16]int
	center2Ctrot  [6435][16]int
	center2Ctprun [6435 * 35 * 2]int
	center2Pmv    = [36]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0}
)

type Center2 struct {
	rl     [8]int
	ct     [16]int
	parity int
}

func newCenter2() *Center2 { return &Center2{} }
func (c *Center2) set(cube *CenterCube, edgeParity int) {
	for i := range c.ct {
		c.ct[i] = cube.ct[i] % 3
	}
	copy(c.rl[:], cube.ct[16:])
	c.parity = edgeParity
}
func (c *Center2) getrl() int {
	idx, r := 0, 4
	for i := 6; i >= 0; i-- {
		if c.rl[i] != c.rl[7] {
			idx += Cnk[i][r]
			r--
		}
	}
	return idx*2 + c.parity
}
func (c *Center2) setrl(idx int) {
	c.parity = idx & 1
	idx >>= 1
	r := 4
	c.rl[7] = 0
	for i := 6; i >= 0; i-- {
		c.rl[i] = 0
		if idx >= Cnk[i][r] {
			idx -= Cnk[i][r]
			r--
			c.rl[i] = 1
		}
	}
}
func (c *Center2) getct() int {
	idx, r := 0, 8
	for i := 14; i >= 0; i-- {
		if c.ct[i] != c.ct[15] {
			idx += Cnk[i][r]
			r--
		}
	}
	return idx
}
func (c *Center2) setct(idx int) {
	r := 8
	c.ct[15] = 0
	for i := 14; i >= 0; i-- {
		c.ct[i] = 0
		if idx >= Cnk[i][r] {
			idx -= Cnk[i][r]
			r--
			c.ct[i] = 1
		}
	}
}
func (c *Center2) rot(r int) {
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
		swap(c.rl[:], 0, 3, 5, 6, 1)
		swap(c.rl[:], 1, 2, 4, 7, 1)
	}
}
func (c *Center2) move(m int) {
	c.parity ^= center2Pmv[m]
	key := m % 3
	switch m / 3 {
	case 0:
		swap(c.ct[:], 0, 1, 2, 3, key)
	case 1:
		swap(c.rl[:], 0, 1, 2, 3, key)
	case 2:
		swap(c.ct[:], 8, 9, 10, 11, key)
	case 3:
		swap(c.ct[:], 4, 5, 6, 7, key)
	case 4:
		swap(c.rl[:], 4, 5, 6, 7, key)
	case 5:
		swap(c.ct[:], 12, 13, 14, 15, key)
	case 6:
		swap(c.ct[:], 0, 1, 2, 3, key)
		swap(c.rl[:], 0, 5, 4, 1, key)
		swap(c.ct[:], 8, 9, 12, 13, key)
	case 7:
		swap(c.rl[:], 0, 1, 2, 3, key)
		swap(c.ct[:], 1, 15, 5, 9, key)
		swap(c.ct[:], 2, 12, 6, 10, key)
	case 8:
		swap(c.ct[:], 8, 9, 10, 11, key)
		swap(c.rl[:], 0, 3, 6, 5, key)
		swap(c.ct[:], 3, 2, 5, 4, key)
	case 9:
		swap(c.ct[:], 4, 5, 6, 7, key)
		swap(c.rl[:], 3, 2, 7, 6, key)
		swap(c.ct[:], 11, 10, 15, 14, key)
	case 10:
		swap(c.rl[:], 4, 5, 6, 7, key)
		swap(c.ct[:], 0, 8, 4, 14, key)
		swap(c.ct[:], 3, 11, 7, 13, key)
	case 11:
		swap(c.ct[:], 12, 13, 14, 15, key)
		swap(c.rl[:], 1, 4, 7, 2, key)
		swap(c.ct[:], 1, 0, 7, 6, key)
	}
}
func center2Init(ctx context.Context) error {
	c := newCenter2()
	for i := 0; i < 70; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for m := 0; m < 28; m++ {
			c.setrl(i)
			c.move(move2std[m])
			center2Rlmv[i][m] = c.getrl()
		}
	}
	for i := 0; i < 70; i++ {
		c.setrl(i)
		for j := 0; j < 16; j++ {
			center2Rlrot[i][j] = c.getrl()
			c.rot(0)
			if j%2 == 1 {
				c.rot(1)
			}
			if j%8 == 7 {
				c.rot(2)
			}
		}
	}
	for i := 0; i < 6435; i++ {
		if i&255 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		c.setct(i)
		for j := 0; j < 16; j++ {
			center2Ctrot[i][j] = c.getct()
			c.rot(0)
			if j%2 == 1 {
				c.rot(1)
			}
			if j%8 == 7 {
				c.rot(2)
			}
		}
	}
	for i := 0; i < 6435; i++ {
		if i&255 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for m := 0; m < 28; m++ {
			c.setct(i)
			c.move(move2std[m])
			center2Ctmv[i][m] = c.getct()
		}
	}
	for i := range center2Ctprun {
		center2Ctprun[i] = -1
	}
	for _, i := range []int{0, 18, 28, 46, 54, 56} {
		center2Ctprun[i] = 0
	}
	depth, done := 0, 6
	for done != len(center2Ctprun) {
		before := done
		for i := range center2Ctprun {
			if i&4095 == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			if center2Ctprun[i] != depth {
				continue
			}
			ct, rl := i/70, i%70
			for m := 0; m < 23; m++ {
				next := center2Ctmv[ct][m]*70 + center2Rlmv[rl][m]
				if center2Ctprun[next] == -1 {
					center2Ctprun[next] = depth + 1
					done++
				}
			}
		}
		if done == before {
			return fmt.Errorf("phase2 pruning did not reach every coordinate")
		}
		depth++
	}
	return ctx.Err()
}
