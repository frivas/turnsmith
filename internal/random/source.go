// Package random defines the versioned deterministic randomness contract.
package random

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
)

type Source interface{ Uint64() uint64 }

func ParseSeed(s string) (uint64, error) {
	if s == "" {
		return 0, fmt.Errorf("--seed must be an unsigned decimal 64-bit integer")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("--seed must be an unsigned decimal 64-bit integer")
		}
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("--seed must be an unsigned decimal 64-bit integer")
	}
	return n, nil
}

func Seed(entropy io.Reader) (uint64, error) {
	var b [8]byte
	if _, err := io.ReadFull(entropy, b[:]); err != nil {
		return 0, fmt.Errorf("acquire root seed: %w", err)
	}
	return binary.BigEndian.Uint64(b[:]), nil
}

// Derive uses length-prefixed UTF-8 strings and big-endian fixed-width integers.
func Derive(root uint64, puzzle, method, version string, index uint64) *rand.PCG {
	h := sha256.New()
	putString := func(s string) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(s)))
		h.Write(n[:])
		h.Write([]byte(s))
	}
	putInt := func(n uint64) { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); h.Write(b[:]) }
	putString("scramble/rng/v1")
	putInt(root)
	putString(puzzle)
	putString(method)
	putString(version)
	putInt(index)
	sum := h.Sum(nil)
	return rand.NewPCG(binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16]))
}

// Below excludes the incomplete residue interval to avoid modulo bias.
// Its behavior is owned here rather than delegated to rand convenience methods.
func Below(rng Source, n uint64) uint64 {
	if n == 0 {
		panic("random.Below: zero bound")
	}
	threshold := -n % n
	for {
		x := rng.Uint64()
		if x >= threshold {
			return x % n
		}
	}
}
