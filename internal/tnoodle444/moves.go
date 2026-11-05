// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Adapted to Go for Turnsmith, 2026, from TNoodle threephase at
// d01a947d9f028f38085cda9b0507a9cf3d3f38a8 (cs.threephase). See COPYING for the GNU GPL terms.

package tnoodle444

const (
	U1  = 0
	U2  = 1
	U3  = 2
	U4  = 3
	U5  = 4
	U6  = 5
	U7  = 6
	U8  = 7
	U9  = 8
	R1  = 9
	R2  = 10
	R3  = 11
	R4  = 12
	R5  = 13
	R6  = 14
	R7  = 15
	R8  = 16
	R9  = 17
	F1  = 18
	F2  = 19
	F3  = 20
	F4  = 21
	F5  = 22
	F6  = 23
	F7  = 24
	F8  = 25
	F9  = 26
	D1  = 27
	D2  = 28
	D3  = 29
	D4  = 30
	D5  = 31
	D6  = 32
	D7  = 33
	D8  = 34
	D9  = 35
	L1  = 36
	L2  = 37
	L3  = 38
	L4  = 39
	L5  = 40
	L6  = 41
	L7  = 42
	L8  = 43
	L9  = 44
	B1  = 45
	B2  = 46
	B3  = 47
	B4  = 48
	B5  = 49
	B6  = 50
	B7  = 51
	B8  = 52
	B9  = 53
	u0  = 0x0
	u1  = 0x1
	u2  = 0x2
	u3  = 0x3
	u4  = 0x4
	u5  = 0x5
	u6  = 0x6
	u7  = 0x7
	u8  = 0x8
	u9  = 0x9
	ua  = 0xa
	ub  = 0xb
	uc  = 0xc
	ud  = 0xd
	ue  = 0xe
	uf  = 0xf
	r0  = 0x10
	r1  = 0x11
	r2  = 0x12
	r3  = 0x13
	r4  = 0x14
	r5  = 0x15
	r6  = 0x16
	r7  = 0x17
	r8  = 0x18
	r9  = 0x19
	ra  = 0x1a
	rb  = 0x1b
	rc  = 0x1c
	rd  = 0x1d
	re  = 0x1e
	rf  = 0x1f
	f0  = 0x20
	f1  = 0x21
	f2  = 0x22
	f3  = 0x23
	f4  = 0x24
	f5  = 0x25
	f6  = 0x26
	f7  = 0x27
	f8  = 0x28
	f9  = 0x29
	fa  = 0x2a
	fb  = 0x2b
	fc  = 0x2c
	fd  = 0x2d
	fe  = 0x2e
	ff  = 0x2f
	d0  = 0x30
	d1  = 0x31
	d2  = 0x32
	d3  = 0x33
	d4  = 0x34
	d5  = 0x35
	d6  = 0x36
	d7  = 0x37
	d8  = 0x38
	d9  = 0x39
	da  = 0x3a
	db  = 0x3b
	dc  = 0x3c
	dd  = 0x3d
	de  = 0x3e
	df  = 0x3f
	l0  = 0x40
	l1  = 0x41
	l2  = 0x42
	l3  = 0x43
	l4  = 0x44
	l5  = 0x45
	l6  = 0x46
	l7  = 0x47
	l8  = 0x48
	l9  = 0x49
	la  = 0x4a
	lb  = 0x4b
	lc  = 0x4c
	ld  = 0x4d
	le  = 0x4e
	lf  = 0x4f
	b0  = 0x50
	b1  = 0x51
	b2  = 0x52
	b3  = 0x53
	b4  = 0x54
	b5  = 0x55
	b6  = 0x56
	b7  = 0x57
	b8  = 0x58
	b9  = 0x59
	ba  = 0x5a
	bb  = 0x5b
	bc  = 0x5c
	bd  = 0x5d
	be  = 0x5e
	bf  = 0x5f
	U   = 0
	R   = 1
	F   = 2
	D   = 3
	L   = 4
	B   = 5
	Ux1 = 0
	Ux2 = 1
	Ux3 = 2
	Rx1 = 3
	Rx2 = 4
	Rx3 = 5
	Fx1 = 6
	Fx2 = 7
	Fx3 = 8
	Dx1 = 9
	Dx2 = 10
	Dx3 = 11
	Lx1 = 12
	Lx2 = 13
	Lx3 = 14
	Bx1 = 15
	Bx2 = 16
	Bx3 = 17
	ux1 = 18
	ux2 = 19
	ux3 = 20
	rx1 = 21
	rx2 = 22
	rx3 = 23
	fx1 = 24
	fx2 = 25
	fx3 = 26
	dx1 = 27
	dx2 = 28
	dx3 = 29
	lx1 = 30
	lx2 = 31
	lx3 = 32
	bx1 = 33
	bx2 = 34
	bx3 = 35
	eom = 36
)

var movesMove2str = [36]string{"U  ", "U2 ", "U' ", "R  ", "R2 ", "R' ", "F  ", "F2 ", "F' ",
	"D  ", "D2 ", "D' ", "L  ", "L2 ", "L' ", "B  ", "B2 ", "B' ",
	"Uw ", "Uw2", "Uw'", "Rw ", "Rw2", "Rw'", "Fw ", "Fw2", "Fw'",
	"Dw ", "Dw2", "Dw'", "Lw ", "Lw2", "Lw'", "Bw ", "Bw2", "Bw'"}
var movesMove2std = [29]int{Ux1, Ux2, Ux3, Rx1, Rx2, Rx3, Fx1, Fx2, Fx3,
	Dx1, Dx2, Dx3, Lx1, Lx2, Lx3, Bx1, Bx2, Bx3,
	ux2, rx1, rx2, rx3, fx2, dx2, lx1, lx2, lx3, bx2, eom}
var movesMove3std = [21]int{Ux1, Ux2, Ux3, Rx2, Fx1, Fx2, Fx3, Dx1, Dx2, Dx3, Lx2, Bx1, Bx2, Bx3,
	ux2, rx2, fx2, dx2, lx2, bx2, eom}

var movesStd2move [37]int
var movesStd3move [37]int
var movesCkmv [37][36]bool
var movesCkmv2 [29][28]bool
var movesCkmv3 [21][20]bool
var movesSkipAxis [36]int
var movesSkipAxis2 [28]int
var movesSkipAxis3 [20]int
var move2str = movesMove2str[:]
var move2std = movesMove2std[:]
var move3std = movesMove3std[:]
var std2move = movesStd2move[:]
var std3move = movesStd3move[:]
var ckmv = movesCkmv[:]
var ckmv2 = movesCkmv2[:]
var ckmv3 = movesCkmv3[:]
var skipAxis = movesSkipAxis[:]
var skipAxis2 = movesSkipAxis2[:]
var skipAxis3 = movesSkipAxis3[:]

func init() {
	for i := 0; i < 29; i++ {
		std2move[move2std[i]] = i
	}
	for i := 0; i < 21; i++ {
		std3move[move3std[i]] = i
	}
	for i := 0; i < 36; i++ {
		for j := 0; j < 36; j++ {
			ckmv[i][j] = i/3 == j/3 || (i/3%3 == j/3%3 && i > j)
		}
	}
	for i := 0; i < 29; i++ {
		for j := 0; j < 28; j++ {
			ckmv2[i][j] = ckmv[move2std[i]][move2std[j]]
		}
	}
	for i := 0; i < 21; i++ {
		for j := 0; j < 20; j++ {
			ckmv3[i][j] = ckmv[move3std[i]][move3std[j]]
		}
	}
	for i := 0; i < 36; i++ {
		skipAxis[i] = 36
		for j := i; j < 36; j++ {
			if !ckmv[i][j] {
				skipAxis[i] = j - 1
				break
			}
		}
	}
	for i := 0; i < 28; i++ {
		skipAxis2[i] = 28
		for j := i; j < 28; j++ {
			if !ckmv2[i][j] {
				skipAxis2[i] = j - 1
				break
			}
		}
	}
	for i := 0; i < 20; i++ {
		skipAxis3[i] = 20
		for j := i; j < 20; j++ {
			if !ckmv3[i][j] {
				skipAxis3[i] = j - 1
				break
			}
		}
	}
}
