# Daily and shared practice sets

```sh
turnsmith daily --date 2026-10-31 --name weekend --output weekend.json
turnsmith daily --puzzle 444 --date 2026-10-31 --count 5 --json
turnsmith guide --file weekend.json --index 1
turnsmith daily --method random-move --length 25 --count 10 --json
```

Daily sets are reproducible practice. They default to the `wca` method and keep
its minimum-distance filter, but use a publicly derived seed rather than direct
cryptographic state sampling. Their result reports `randomness: seeded-practice`.
Do not use predictable daily seeds for secret competition scrambles. Official
competitions require the current [official WCA scramble program](https://www.worldcubeassociation.org/regulations/#4b).

The default count is five. Dates must be valid `YYYY-MM-DD` calendar dates; an
omitted date uses today's UTC date. A manifest records its name, date, UTC policy,
seed-policy version, root seed, puzzle, method, algorithm version, randomness
mode, and actual moves. Saved moves remain replayable after a generator changes.
`--output PATH` saves a JSON manifest after complete generation and validation,
using a sibling temporary file and replacement. `--json` also selects JSON for
stdout. Manifests are bounded at 1 MiB to match inspection's input limit.

## Seed policy version 1

`internal/daily.Build` hashes five length-prefixed UTF-8 fields with SHA-256:
`turnsmith/daily`, `1`, the UTC calendar date, canonical puzzle ID, and method ID.
Field lengths are unsigned big-endian 32-bit integers. The first eight digest
bytes form the unsigned big-endian root seed. Aliases resolve before hashing.
Names, counts, and lengths do not participate. Increasing count preserves the
previous scramble prefix for the same method and algorithm version. A different
method changes the seed. Algorithm versions also participate in each scramble's
normal stream derivation and are recorded in the result.

For example, date `2026-10-31`, puzzle `333`, and explicit method `random-move`
produce root seed `14884999549874360842`. This example does not select the default
`wca` method. `--seed` is unavailable in daily mode; use ordinary generation to
choose a seed directly.

## Puzzle methods

Default `wca` supports 222, 333, and 444. It chooses its move count and rejects
`--length`. The 444 method requires the separate native Go `turnsmith-wca`
companion included in release packages. Install it beside the main executable
or on PATH. No Java or runtime download is required. The backend constructs its
tables lazily in memory; initialization/search failure is an error, without a
random-move fallback.

Legacy methods remain explicit and preserve algorithm version 1:

- `222/random-state` and `333/random-state` retain their original sampling.
- `333/random-move` defaults to 25 moves.
- `444/random-move` defaults to 40 moves and uses one- or two-layer turns,
  including `Uw`, `Rw2`, and `Fw'`.

The random-move methods accept lengths from 1 to 1000, exclude adjacent same-face
moves and A-opposite(A)-A windows, and make no uniform-state claim. Their aliases
include `3x3`, `3x3x3`, `4x4`, and `4x4x4`. Names and output formatting do not alter
a seeded stream.
