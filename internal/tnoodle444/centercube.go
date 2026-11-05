// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Adapted to Go for Turnsmith, 2026, from TNoodle threephase at
// d01a947d9f028f38085cda9b0507a9cf3d3f38a8 (cs.threephase). See COPYING for the GNU GPL terms.

package tnoodle444

import "github.com/frivas/turnsmith/internal/random"

type CenterCube struct{ ct [24]int }

var centerCubeCenter333Map = [6]int{0, 4, 2, 1, 5, 3}

func newCenterCube() *CenterCube {
	c := &CenterCube{}
	for i := range c.ct {
		c.ct[i] = fullCubeCenterFacelet[i] / 16
	}
	return c
}
func (c *CenterCube) fill333Facelet(facelet []byte) {
	for i := 0; i < 6; i++ {
		idx := centerCubeCenter333Map[i] << 2
		if c.ct[idx] != c.ct[idx+1] || c.ct[idx+1] != c.ct[idx+2] || c.ct[idx+2] != c.ct[idx+3] {
			panic("unsolved center")
		}
		facelet[4+i*9] = "URFDLB"[c.ct[idx]]
	}
}
func (c *CenterCube) copy(other *CenterCube) { c.ct = other.ct }
func newCenterCubeRandom(r random.Source) *CenterCube {
	c := newCenterCube()
	for i := 0; i < 23; i++ {
		t := i + int(random.Below(r, uint64(24-i)))
		c.ct[i], c.ct[t] = c.ct[t], c.ct[i]
	}
	return c
}
func (c *CenterCube) move(m int) {
	key := m % 3
	m /= 3
	switch m {
	case 0: //U
		swap(c.ct[:], 0, 1, 2, 3, key)

	case 1: //R
		swap(c.ct[:], 16, 17, 18, 19, key)

	case 2: //F
		swap(c.ct[:], 8, 9, 10, 11, key)

	case 3: //D
		swap(c.ct[:], 4, 5, 6, 7, key)

	case 4: //L
		swap(c.ct[:], 20, 21, 22, 23, key)

	case 5: //B
		swap(c.ct[:], 12, 13, 14, 15, key)

	case 6: //u
		swap(c.ct[:], 0, 1, 2, 3, key)
		swap(c.ct[:], 8, 20, 12, 16, key)
		swap(c.ct[:], 9, 21, 13, 17, key)

	case 7: //r
		swap(c.ct[:], 16, 17, 18, 19, key)
		swap(c.ct[:], 1, 15, 5, 9, key)
		swap(c.ct[:], 2, 12, 6, 10, key)

	case 8: //f
		swap(c.ct[:], 8, 9, 10, 11, key)
		swap(c.ct[:], 2, 19, 4, 21, key)
		swap(c.ct[:], 3, 16, 5, 22, key)

	case 9: //d
		swap(c.ct[:], 4, 5, 6, 7, key)
		swap(c.ct[:], 10, 18, 14, 22, key)
		swap(c.ct[:], 11, 19, 15, 23, key)

	case 10: //l
		swap(c.ct[:], 20, 21, 22, 23, key)
		swap(c.ct[:], 0, 8, 4, 14, key)
		swap(c.ct[:], 3, 11, 7, 13, key)

	case 11: //b
		swap(c.ct[:], 12, 13, 14, 15, key)
		swap(c.ct[:], 1, 20, 7, 18, key)
		swap(c.ct[:], 0, 23, 6, 17, key)

	}
}
