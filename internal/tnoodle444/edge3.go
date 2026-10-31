// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Go adaptation of TNoodle threephase, pinned d01a947d9f028f38085cda9b0507a9cf3d3f38a8, by Francisco Rivas and contributors.
// This file is part of the separately licensed turnsmith-wca backend.
package tnoodle444

import "context"

const edge3NSym = 1538
const edge3NRaw = 20160
const edge3NEprun = edge3NSym * edge3NRaw
const edge3MaxDepth = 10

var edge3Eprun = make([]uint32, edge3NEprun/16)
var edge3Sym2raw [edge3NSym]int
var edge3Symstate [edge3NSym]uint16
var edge3Raw2sym [11880]int
var edge3Syminv = [8]int{0, 1, 6, 3, 4, 5, 2, 7}
var edge3Mvrot [160][12]int
var edge3Mvroto [160][12]int
var edge3FactX = [13]int{1, 1, 1, 3, 12, 60, 360, 2520, 20160, 181440, 1814400, 19958400, 239500800}
var edge3FullEdgeMap = [12]int{0, 2, 4, 6, 1, 3, 7, 5, 8, 9, 10, 11}

type Edge3 struct {
	edge, edgeo, temp [12]int
	isStd             bool
}

func newEdge3() *Edge3 {
	e := &Edge3{isStd: true}
	for i := range e.edgeo {
		e.edgeo[i] = i
	}
	return e
}

func edge3InitMvrot(ctx context.Context) error {
	e := newEdge3()
	for m := 0; m < 20; m++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for r := 0; r < 8; r++ {
			e.set(0)
			e.move(m)
			e.rotate(r)
			edge3Mvrot[m<<3|r] = e.edge
			e.std()
			edge3Mvroto[m<<3|r] = e.temp
		}
	}
	return nil
}
func edge3InitRaw2Sym(ctx context.Context) error {
	e := newEdge3()
	var occ [11880 / 8]uint8
	count := 0
	for i := 0; i < 11880; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if occ[i>>3]&(1<<uint(i&7)) != 0 {
			continue
		}
		e.set(i * edge3FactX[8])
		for j := 0; j < 8; j++ {
			idx := e.get(4)
			if idx == i {
				edge3Symstate[count] |= 1 << uint(j)
			}
			occ[idx>>3] |= 1 << uint(idx&7)
			edge3Raw2sym[idx] = count<<3 | edge3Syminv[j]
			e.rot(0)
			if j%2 == 1 {
				e.rot(1)
				e.rot(2)
			}
		}
		edge3Sym2raw[count] = i
		count++
	}
	if count != edge3NSym {
		panic("edge3 symmetry count mismatch")
	}
	return nil
}
func edge3SetPruning(table []uint32, index, value int) {
	table[index>>4] ^= uint32(3^value) << uint((index&15)<<1)
}
func edge3GetPruning(table []uint32, index int) int {
	return int(table[index>>4] >> uint((index&15)<<1) & 3)
}
func edge3Getprun(edge int, previous ...int) int {
	depm3 := edge3GetPruning(edge3Eprun, edge)
	if depm3 == 3 {
		return edge3MaxDepth
	}
	if len(previous) > 0 {
		prun := previous[0]
		return (depm3-prun+16)%3 + prun - 1
	}
	e := newEdge3()
	depth := 0
	for edge != 0 {
		if depm3 == 0 {
			depm3 = 2
		} else {
			depm3--
		}
		cord1 := edge3Sym2raw[edge/edge3NRaw]
		cord2 := edge % edge3NRaw
		e.set(cord1*edge3NRaw + cord2)
		found := false
		for m := 0; m < 17; m++ {
			cord1x := edge3Getmvrot(e.edge[:], m<<3, 4)
			symcord1x := edge3Raw2sym[cord1x]
			symx := symcord1x & 7
			symcord1x >>= 3
			cord2x := edge3Getmvrot(e.edge[:], m<<3|symx, 10) % edge3NRaw
			idx := symcord1x*edge3NRaw + cord2x
			if edge3GetPruning(edge3Eprun, idx) == depm3 {
				depth++
				edge = idx
				found = true
				break
			}
		}
		if !found {
			panic("edge3 pruning table has no descending path")
		}
	}
	return depth
}
func edge3CreatePrun(ctx context.Context) error {
	e, f, g := newEdge3(), newEdge3(), newEdge3()
	for i := range edge3Eprun {
		edge3Eprun[i] = ^uint32(0)
	}
	done := 1
	edge3SetPruning(edge3Eprun, 0, 0)
	for depth := 0; done != edge3NEprun && depth < edge3MaxDepth-1; depth++ {
		depm3 := depth % 3
		dep1m3 := (depth + 1) % 3
		for base := 0; base < edge3NEprun; base += 16 {
			if base&0x3ffff == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			value := edge3Eprun[base>>4]
			if value == ^uint32(0) {
				continue
			}
			for i, end := base, base+16; i < end; i, value = i+1, value>>2 {
				if int(value&3) != depm3 {
					continue
				}
				cord1 := edge3Sym2raw[i/edge3NRaw]
				cord2 := i % edge3NRaw
				e.set(cord1*edge3NRaw + cord2)
				for m := 0; m < 17; m++ {
					cord1x := edge3Getmvrot(e.edge[:], m<<3, 4)
					symcord1x := edge3Raw2sym[cord1x]
					symx := symcord1x & 7
					symcord1x >>= 3
					cord2x := edge3Getmvrot(e.edge[:], m<<3|symx, 10) % edge3NRaw
					idx := symcord1x*edge3NRaw + cord2x
					if edge3GetPruning(edge3Eprun, idx) != 3 {
						continue
					}
					edge3SetPruning(edge3Eprun, idx, dep1m3)
					done++
					symState := edge3Symstate[symcord1x]
					if symState == 1 {
						continue
					}
					f.copy(e)
					f.move(m)
					f.rotate(symx)
					for j := 1; ; j++ {
						symState >>= 1
						if symState == 0 {
							break
						}
						if symState&1 != 1 {
							continue
						}
						g.copy(f)
						g.rotate(j)
						other := symcord1x*edge3NRaw + g.get(10)%edge3NRaw
						if edge3GetPruning(edge3Eprun, other) == 3 {
							edge3SetPruning(edge3Eprun, other, dep1m3)
							done++
						}
					}
				}
			}
		}
	}
	return nil
}
func (e *Edge3) getsym() int {
	cord1 := e.get(4)
	symcord1 := edge3Raw2sym[cord1]
	sym := symcord1 & 7
	symcord1 >>= 3
	e.rotate(sym)
	cord2 := e.get(10) % edge3NRaw
	return symcord1*edge3NRaw + cord2
}
func (e *Edge3) setEdge(c *EdgeCube) int {
	for i := range e.edgeo {
		e.edgeo[i] = i
	}
	e.isStd = true
	for i := 0; i < 12; i++ {
		e.temp[i] = i
		e.edge[i] = c.ep[edge3FullEdgeMap[i]+12] % 12
	}
	parity := 1
	for i := 0; i < 12; i++ {
		for e.edge[i] != i {
			t := e.edge[i]
			e.edge[i] = e.edge[t]
			e.edge[t] = t
			e.temp[i], e.temp[t] = e.temp[t], e.temp[i]
			parity ^= 1
		}
	}
	for i := 0; i < 12; i++ {
		e.edge[i] = e.temp[c.ep[edge3FullEdgeMap[i]]%12]
	}
	return parity
}
func (e *Edge3) copy(other *Edge3) { e.edge = other.edge; e.edgeo = other.edgeo; e.isStd = other.isStd }
func edge3Getmvrot(ep []int, mrIdx, end int) int {
	movo, mov := &edge3Mvroto[mrIdx], &edge3Mvrot[mrIdx]
	idx := 0
	value := uint64(0xba9876543210)
	for i := 0; i < end; i++ {
		shift := uint(movo[ep[mov[i]]] << 2)
		idx *= 12 - i
		idx += int(value >> shift & 15)
		value -= uint64(0x111111111110) << shift
	}
	return idx
}
func (e *Edge3) std() {
	for i := 0; i < 12; i++ {
		e.temp[e.edgeo[i]] = i
	}
	for i := 0; i < 12; i++ {
		e.edge[i] = e.temp[e.edge[i]]
		e.edgeo[i] = i
	}
	e.isStd = true
}
func (e *Edge3) get(end int) int {
	if !e.isStd {
		e.std()
	}
	idx := 0
	value := uint64(0xba9876543210)
	for i := 0; i < end; i++ {
		shift := uint(e.edge[i] << 2)
		idx *= 12 - i
		idx += int(value >> shift & 15)
		value -= uint64(0x111111111110) << shift
	}
	return idx
}
func (e *Edge3) set(idx int) {
	value := uint64(0xba9876543210)
	parity := 0
	for i := 0; i < 11; i++ {
		p := edge3FactX[11-i]
		v := idx / p
		idx %= p
		parity ^= v
		shift := uint(v << 2)
		e.edge[i] = int(value >> shift & 15)
		mask := (uint64(1) << shift) - 1
		value = (value & mask) + ((value >> 4) &^ mask)
	}
	if parity&1 == 0 {
		e.edge[11] = int(value)
	} else {
		e.edge[11] = e.edge[10]
		e.edge[10] = int(value)
	}
	for i := 0; i < 12; i++ {
		e.edgeo[i] = i
	}
	e.isStd = true
}
func (e *Edge3) move(i int) {
	e.isStd = false
	switch i {
	case 0: //U
		edge3Circle(e.edge[:], 0, 4, 1, 5)
		edge3Circle(e.edgeo[:], 0, 4, 1, 5)

	case 1: //U2
		edge3Swap4(e.edge[:], 0, 4, 1, 5)
		edge3Swap4(e.edgeo[:], 0, 4, 1, 5)

	case 2: //U'
		edge3Circle(e.edge[:], 0, 5, 1, 4)
		edge3Circle(e.edgeo[:], 0, 5, 1, 4)

	case 3: //R2
		edge3Swap4(e.edge[:], 5, 10, 6, 11)
		edge3Swap4(e.edgeo[:], 5, 10, 6, 11)

	case 4: //F
		edge3Circle(e.edge[:], 0, 11, 3, 8)
		edge3Circle(e.edgeo[:], 0, 11, 3, 8)

	case 5: //F2
		edge3Swap4(e.edge[:], 0, 11, 3, 8)
		edge3Swap4(e.edgeo[:], 0, 11, 3, 8)

	case 6: //F'
		edge3Circle(e.edge[:], 0, 8, 3, 11)
		edge3Circle(e.edgeo[:], 0, 8, 3, 11)

	case 7: //D
		edge3Circle(e.edge[:], 2, 7, 3, 6)
		edge3Circle(e.edgeo[:], 2, 7, 3, 6)

	case 8: //D2
		edge3Swap4(e.edge[:], 2, 7, 3, 6)
		edge3Swap4(e.edgeo[:], 2, 7, 3, 6)

	case 9: //D'
		edge3Circle(e.edge[:], 2, 6, 3, 7)
		edge3Circle(e.edgeo[:], 2, 6, 3, 7)

	case 10: //L2
		edge3Swap4(e.edge[:], 4, 8, 7, 9)
		edge3Swap4(e.edgeo[:], 4, 8, 7, 9)

	case 11: //B
		edge3Circle(e.edge[:], 1, 9, 2, 10)
		edge3Circle(e.edgeo[:], 1, 9, 2, 10)

	case 12: //B2
		edge3Swap4(e.edge[:], 1, 9, 2, 10)
		edge3Swap4(e.edgeo[:], 1, 9, 2, 10)

	case 13: //B'
		edge3Circle(e.edge[:], 1, 10, 2, 9)
		edge3Circle(e.edgeo[:], 1, 10, 2, 9)

	case 14: //u2
		edge3Swap4(e.edge[:], 0, 4, 1, 5)
		edge3Swap4(e.edgeo[:], 0, 4, 1, 5)
		edge3Swap2(e.edge[:], 9, 11)
		edge3Swap2(e.edgeo[:], 8, 10)

	case 15: //r2
		edge3Swap4(e.edge[:], 5, 10, 6, 11)
		edge3Swap4(e.edgeo[:], 5, 10, 6, 11)
		edge3Swap2(e.edge[:], 1, 3)
		edge3Swap2(e.edgeo[:], 0, 2)

	case 16: //f2
		edge3Swap4(e.edge[:], 0, 11, 3, 8)
		edge3Swap4(e.edgeo[:], 0, 11, 3, 8)
		edge3Swap2(e.edge[:], 5, 7)
		edge3Swap2(e.edgeo[:], 4, 6)

	case 17: //d2
		edge3Swap4(e.edge[:], 2, 7, 3, 6)
		edge3Swap4(e.edgeo[:], 2, 7, 3, 6)
		edge3Swap2(e.edge[:], 8, 10)
		edge3Swap2(e.edgeo[:], 9, 11)

	case 18: //l2
		edge3Swap4(e.edge[:], 4, 8, 7, 9)
		edge3Swap4(e.edgeo[:], 4, 8, 7, 9)
		edge3Swap2(e.edge[:], 0, 2)
		edge3Swap2(e.edgeo[:], 1, 3)

	case 19: //b2
		edge3Swap4(e.edge[:], 1, 9, 2, 10)
		edge3Swap4(e.edgeo[:], 1, 9, 2, 10)
		edge3Swap2(e.edge[:], 4, 6)
		edge3Swap2(e.edgeo[:], 5, 7)

	}
}

func (e *Edge3) rot(r int) {
	e.isStd = false
	switch r {
	case 0:
		e.move(14)
		e.move(17)

	case 1:
		e.circlex(11, 5, 10, 6) //r
		e.circlex(5, 10, 6, 11)
		e.circlex(1, 2, 3, 0)
		e.circlex(4, 9, 7, 8) //l'
		e.circlex(8, 4, 9, 7)
		e.circlex(0, 1, 2, 3)

	case 2:
		e.swapx(4, 5)
		e.swapx(5, 4)
		e.swapx(11, 8)
		e.swapx(8, 11)
		e.swapx(7, 6)
		e.swapx(6, 7)
		e.swapx(9, 10)
		e.swapx(10, 9)
		e.swapx(1, 1)
		e.swapx(0, 0)
		e.swapx(3, 3)
		e.swapx(2, 2)

	}
}

func (e *Edge3) rotate(r int) {
	for r >= 2 {
		r -= 2
		e.rot(1)
		e.rot(2)
	}
	if r != 0 {
		e.rot(0)
	}
}

func edge3Circle(arr []int, a, b, c, d int) {
	arr[a], arr[b], arr[c], arr[d] = arr[d], arr[a], arr[b], arr[c]
}
func edge3Swap4(arr []int, a, b, c, d int) {
	arr[a], arr[c] = arr[c], arr[a]
	arr[b], arr[d] = arr[d], arr[b]
}
func edge3Swap2(arr []int, a, b int) { arr[a], arr[b] = arr[b], arr[a] }
func (e *Edge3) swapx(x, y int)      { e.edge[x], e.edgeo[y] = e.edgeo[y], e.edge[x] }
func (e *Edge3) circlex(a, b, c, d int) {
	temp := e.edgeo[d]
	e.edgeo[d] = e.edge[c]
	e.edge[c] = e.edgeo[b]
	e.edgeo[b] = e.edge[a]
	e.edge[a] = temp
}
