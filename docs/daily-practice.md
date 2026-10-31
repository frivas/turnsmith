# Daily and shared practice sets

`internal/daily.Build` creates a portable practice-set manifest around the normal generation result. Its date is a strict `YYYY-MM-DD` calendar date. An omitted date uses today's UTC date. A set records its name, date, UTC policy, seed policy version, root seed, puzzle, method, algorithm version and each scramble.

Seed policy version 1 hashes five length-prefixed UTF-8 fields with SHA-256: `turnsmith/daily`, `1`, the calendar date, the canonical puzzle ID and the method ID. Field lengths use unsigned big-endian 32-bit integers. The first eight digest bytes, interpreted as an unsigned big-endian integer, are the root seed. Puzzle aliases resolve before hashing. Names, counts and lengths do not participate. Changing a name preserves a shared set, while increasing count extends the existing scramble prefix. Algorithm versions participate in the normal per-scramble stream derivation and remain recorded in the manifest.

For example, the date `2026-10-31`, puzzle `333` and method `random-move` produce root seed `14884999549874360842`. The date means a UTC calendar day regardless of the user's local timezone. These predictable public seeds are for shared practice, not secret competition scrambles. Saved manifests retain their actual sequences when future algorithm versions change.

# Four-layer cube generation

`internal/cube444.Registration()` provides puzzle `444`, aliases `4x4` and `4x4x4`, and random-move algorithm version 1. It defaults to 40 moves and permits 1–1000 moves. Every step chooses an admissible face, a width of one or two layers, and a quarter, half or inverse turn using the caller's deterministic random source. Two-layer notation includes `Uw`, `Rw2` and `Fw'`. The generator rejects adjacent same-face moves and A-opposite(A)-A windows regardless of width. It generates actual outer and wide-layer turns; it does not claim uniform reachable-state sampling or competition approval.
