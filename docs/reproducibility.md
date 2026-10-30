# Reproducibility contract

This document defines `333/random-move`, algorithm version `1`, and stream
domain `scramble/rng/v1`. JSON schema version 1 is a separate compatibility
contract and does not determine the generator's sequence.

## Seed and stream encoding

An explicit seed is an unsigned decimal integer from 0 through 2^64−1.
Leading zeros are accepted and normalized in JSON. Signs, whitespace, hex,
fractional notation, empty strings, and overflow are errors. Without a supplied
seed, read exactly eight bytes from the OS cryptographic random source and
interpret them as a big-endian unsigned 64-bit integer. Failure is fatal, with
no clock fallback. This random source is injected at the application boundary.

For each zero-based batch index, hash the following concatenation with SHA-256,
in exactly this order:

1. String `scramble/rng/v1`.
2. Root seed as an unsigned 64-bit big-endian integer.
3. Canonical puzzle string, e.g. `333`.
4. Method ID string, e.g. `random-move`.
5. Algorithm version string, e.g. `1`.
6. Index as an unsigned 64-bit big-endian integer.

Every string is encoded as an unsigned 32-bit big-endian UTF-8 byte count,
followed by those bytes. The domain is length-prefixed too. There are no
terminators, padding, delimiters, or decimal-string integers. Length is not a
stream-identity field; a longer scramble extends the shorter one's sequence.
Count and presentation format are also absent.

Initialize `math/rand/v2.NewPCG(seed1, seed2)` with digest bytes [0:8] and [8:16],
each interpreted as a big-endian unsigned 64-bit integer. Do not use the
remaining digest bytes. PCG initialization directly sets its high and low state
words to these values, with no warm-up draws.

For seed 42, puzzle `333`, method `random-move`, algorithm `1`, index 0, the full
digest is:

```text
e170e3277097fce01678198488b970d0eb590b952ea4ac570db5c1759a9864d5
```

The first five raw PCG outputs are:

```text
5067026879568920168
18189984400193564563
728703022228311301
3919567004198634860
1278838995454122105
```

## PCG arithmetic

The standard-library PCG uses a 128-bit state S, initially `(seed1 << 64) | seed2`.
Before each output, set S to `(S * M + I) modulo 2^128`, where:

```text
M = (2549297995355413924 << 64) | 4865540595714422341
I = (6364136223846793005 << 64) | 1442695040888963407
```

Let H and L be the resulting high and low 64-bit words. Apply the DXSM output
transformation, with multiplication modulo 2^64 and logical right shifts:

```text
H = H xor (H >> 32)
H = H * 0xda942042e4dd58b5
H = H xor (H >> 48)
output = H * (L | 1)
```

This pins the raw-word behavior even if a future Go toolchain changes its PCG
implementation. See Go and PCG attribution in [third-party notices](../THIRD_PARTY_NOTICES.md).

## Bounded selection

To obtain an unbiased integer below n, compute `threshold = 2^64 mod n`.
Read raw PCG words until x is at least threshold, then return `x mod n`.
In unsigned Go arithmetic the threshold is `-n % n`. Consume one bounded draw
for face selection, then one for turn selection. Enumerate candidates and turns
in the order in [algorithms](algorithms.md). Do not substitute a different
bounded-selection API even if it is also unbiased: different consumption can
change every subsequent move.

## Vectors and compatibility

The fixed fixtures in [vectors.json](../internal/cli/testdata/vectors.json)
cover seeds 0, 42, and the maximum seed; lengths 1, 2, 25, and 30; and multiple
indices. They were independently calculated using Python integer arithmetic
and compared with the Go executable. The tests also pin raw PCG outputs.

For `scramble --seed 42`, the sequence is:

```text
L2 D' U F L' B' D' L2 U2 D' F2 R F L2 U L2 B2 U L2 D L2 R2 U2 B' L
```

A batch of three is the prefix of a batch of ten. Puzzle aliases resolve before
hashing. JSON and text serialize the same results. Every invocation and every
scramble owns its stream, options, and working history. There is no shared RNG.

Algorithm-version changes are required for intentional changes to candidate
order, turn order, rules, stream encoding, bounded draws, PRNG behavior, or
other changes to fixed output. Review failed vectors before changing them.
Preserve old behavior or increment the algorithm version, record the reason in
release notes, and publish new fixtures. A version bump does not imply the CLI
retains old generator implementations; replay with the matching released binary.
Seeds alone do not guarantee reproduction across algorithm versions.

Schema version 1 keeps seed strings, canonical IDs, reproduction metadata,
optional typed length, move tokens, joined sequences, token counts, and one-based
user-facing indices compatible. Consumers should tolerate new object fields.
No timestamps or host-dependent paths are included.
