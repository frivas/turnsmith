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
	center3Ctmove [35 * 35 * 12 * 2][20]int
	center3Pmove  = [20]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1}
	center3Prun   [35 * 35 * 12 * 2]int
	center3Rl2std = [12]int{0, 9, 14, 23, 27, 28, 41, 42, 46, 55, 60, 69}
	center3Std2rl [70]int
)

type Center3 struct {
	ud, rl, fb [8]int
	parity     int
}

func newCenter3() *Center3 { return &Center3{} }
func (c *Center3) set(cube *CenterCube, eXcParity int) {
	// Java's chained boolean XOR is odd-parity XOR, not integer comparison.
	a, b, d := cube.ct[0]%3, cube.ct[8]%3, cube.ct[16]%3
	p := 1
	if ((a > b) != (b > d)) != (a > d) {
		p = 0
	}
	for i := 0; i < 8; i++ {
		c.ud[i] = (cube.ct[i] / 3) ^ 1
		c.fb[i] = (cube.ct[i+8] / 3) ^ 1
		c.rl[i] = (cube.ct[i+16] / 3) ^ 1 ^ p
	}
	c.parity = p ^ eXcParity
}
func (c *Center3) getct() int {
	idx, r := 0, 4
	for i := 6; i >= 0; i-- {
		if c.ud[i] != c.ud[7] {
			idx += Cnk[i][r]
			r--
		}
	}
	idx *= 35
	r = 4
	for i := 6; i >= 0; i-- {
		if c.fb[i] != c.fb[7] {
			idx += Cnk[i][r]
			r--
		}
	}
	idx *= 12
	check := c.fb[7] ^ c.ud[7]
	idxrl := 0
	r = 4
	for i := 7; i >= 0; i-- {
		if c.rl[i] != check {
			idxrl += Cnk[i][r]
			r--
		}
	}
	return c.parity + 2*(idx+center3Std2rl[idxrl])
}
func (c *Center3) setct(idx int) {
	c.parity = idx & 1
	idx >>= 1
	idxrl := center3Rl2std[idx%12]
	idx /= 12
	r := 4
	for i := 7; i >= 0; i-- {
		c.rl[i] = 0
		if idxrl >= Cnk[i][r] {
			idxrl -= Cnk[i][r]
			r--
			c.rl[i] = 1
		}
	}
	idxfb := idx % 35
	idx /= 35
	r = 4
	c.fb[7] = 0
	for i := 6; i >= 0; i-- {
		c.fb[i] = 0
		if idxfb >= Cnk[i][r] {
			idxfb -= Cnk[i][r]
			r--
			c.fb[i] = 1
		}
	}
	r = 4
	c.ud[7] = 0
	for i := 6; i >= 0; i-- {
		c.ud[i] = 0
		if idx >= Cnk[i][r] {
			idx -= Cnk[i][r]
			r--
			c.ud[i] = 1
		}
	}
}
func (c *Center3) move(i int) {
	c.parity ^= center3Pmove[i]
	switch i {
	case 0, 1, 2:
		swap(c.ud[:], 0, 1, 2, 3, i%3)
	case 3:
		swap(c.rl[:], 0, 1, 2, 3, 1)
	case 4, 5, 6:
		swap(c.fb[:], 0, 1, 2, 3, (i-1)%3)
	case 7, 8, 9:
		swap(c.ud[:], 4, 5, 6, 7, (i-1)%3)
	case 10:
		swap(c.rl[:], 4, 5, 6, 7, 1)
	case 11, 12, 13:
		swap(c.fb[:], 4, 5, 6, 7, (i+1)%3)
	case 14:
		swap(c.ud[:], 0, 1, 2, 3, 1)
		swap(c.rl[:], 0, 5, 4, 1, 1)
		swap(c.fb[:], 0, 5, 4, 1, 1)
	case 15:
		swap(c.rl[:], 0, 1, 2, 3, 1)
		swap(c.fb[:], 1, 4, 7, 2, 1)
		swap(c.ud[:], 1, 6, 5, 2, 1)
	case 16:
		swap(c.fb[:], 0, 1, 2, 3, 1)
		swap(c.ud[:], 3, 2, 5, 4, 1)
		swap(c.rl[:], 0, 3, 6, 5, 1)
	case 17:
		swap(c.ud[:], 4, 5, 6, 7, 1)
		swap(c.rl[:], 3, 2, 7, 6, 1)
		swap(c.fb[:], 3, 2, 7, 6, 1)
	case 18:
		swap(c.rl[:], 4, 5, 6, 7, 1)
		swap(c.fb[:], 0, 3, 6, 5, 1)
		swap(c.ud[:], 0, 3, 4, 7, 1)
	case 19:
		swap(c.fb[:], 4, 5, 6, 7, 1)
		swap(c.ud[:], 0, 7, 6, 1, 1)
		swap(c.rl[:], 1, 4, 7, 2, 1)
	}
}
func center3Init(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, v := range center3Rl2std {
		center3Std2rl[v] = i
	}
	c := newCenter3()
	for i := range center3Ctmove {
		if i&255 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for m := 0; m < 20; m++ {
			c.setct(i)
			c.move(m)
			center3Ctmove[i][m] = c.getct()
		}
	}
	for i := range center3Prun {
		center3Prun[i] = -1
	}
	center3Prun[0] = 0
	depth, done := 0, 1
	for done != len(center3Prun) {
		before := done
		for i := range center3Prun {
			if i&1023 == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			if center3Prun[i] != depth {
				continue
			}
			for m := 0; m < 17; m++ {
				next := center3Ctmove[i][m]
				if center3Prun[next] == -1 {
					center3Prun[next] = depth + 1
					done++
				}
			}
		}
		if done == before {
			return fmt.Errorf("phase3 pruning did not reach every coordinate")
		}
		depth++
	}
	return ctx.Err()
}
