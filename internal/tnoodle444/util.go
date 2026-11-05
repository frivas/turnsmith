// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Adapted to Go for Turnsmith, 2026, from TNoodle threephase at
// d01a947d9f028f38085cda9b0507a9cf3d3f38a8 (cs.threephase). See COPYING for the GNU GPL terms.
package tnoodle444

var Cnk [25][25]int
var fact [13]int

func init() {
	for i := 0; i < 25; i++ {
		Cnk[i][i] = 1
		Cnk[i][0] = 1
	}
	for i := 1; i < 25; i++ {
		for j := 1; j <= i; j++ {
			Cnk[i][j] = Cnk[i-1][j] + Cnk[i-1][j-1]
		}
	}
	fact[0] = 1
	for i := 0; i < 12; i++ {
		fact[i+1] = fact[i] * (i + 1)
	}
}
func utilTomove(s string) []int {
	ret := []int{}
	axis := -1
	for _, r := range s {
		switch r {
		case 'U':
			axis = 0
		case 'R':
			axis = 3
		case 'F':
			axis = 6
		case 'D':
			axis = 9
		case 'L':
			axis = 12
		case 'B':
			axis = 15
		case 'u':
			axis = 18
		case 'r':
			axis = 21
		case 'f':
			axis = 24
		case 'd':
			axis = 27
		case 'l':
			axis = 30
		case 'b':
			axis = 33
		case ' ':
			if axis != -1 {
				ret = append(ret, axis)
			}
			axis = -1
		case '2':
			axis++
		case '\'':
			axis += 2
		case 'w':
			axis += 18
		}
	}
	if axis != -1 {
		ret = append(ret, axis)
	}
	return ret
}
func swap(arr []int, a, b, c, d, key int) {
	switch key {
	case 0:
		arr[a], arr[b], arr[c], arr[d] = arr[d], arr[a], arr[b], arr[c]
	case 1:
		arr[a], arr[b], arr[c], arr[d] = arr[c], arr[d], arr[a], arr[b]
	case 2:
		arr[a], arr[b], arr[c], arr[d] = arr[b], arr[c], arr[d], arr[a]
	}
}
func set8Perm(arr []int, idx int) {
	val := uint64(0x76543210)
	for i := 0; i < 7; i++ {
		p := fact[7-i]
		v := idx / p
		idx -= v * p
		shift := uint(v << 2)
		arr[i] = int((val >> shift) & 15)
		mask := (uint64(1) << shift) - 1
		val = (val & mask) + ((val >> 4) &^ mask)
	}
	arr[7] = int(val)
}
func parity(arr []int) int {
	p := 0
	for i := range arr {
		for j := i; j < len(arr); j++ {
			if arr[i] > arr[j] {
				p ^= 1
			}
		}
	}
	return p
}
