# turnsmith

Scrambles for physical twisty-puzzle practice.

## Language

**Puzzle**:
A particular twisty-puzzle type, identified independently of any generation method.

**Method**:
An algorithm for producing scrambles for a puzzle, with its own version and guarantees.

**Scramble**:
An ordered sequence of notation tokens intended to mix a physical puzzle.

**Move**:
A puzzle operation represented by one emitted token. A half turn is one move.

**Root seed**:
The unsigned number identifying a reproducible practice batch for a given method
version and its parameters.

**Batch**:
An ordered collection of independently generated scrambles. Seeded batches share
a root seed; cryptographic `wca` batches draw state randomness directly from OS
entropy and report no root seed.

**Random-move generation**:
Selection of legal moves subject to local redundancy restrictions. It does not
imply uniform sampling of reachable puzzle states.

**Random-state generation**:
Sampling a reachable state and constructing a scramble that reaches it. The
sampling space and solver guarantee belong to the method version.

**Algorithm**:
A supplied move sequence for inspection or repeated practice.

**Cube state**:
The colored stickers after a move prefix, starting solved in white-up, green-front orientation.

**Solve**:
One recorded timed attempt, with its scramble, raw duration, penalty and optional notes.

**Session**:
A named collection of solves whose statistics are calculated together.

**Training case**:
A named user algorithm with its inverse setup, learning status and recorded attempts.

**Practice set**:
A shareable batch with a UTC practice date, versioned seed policy and generation metadata.


**Competition-style generation**:
The `wca` method, default for 222, 333, and 444. It samples random states with
minimum solving-distance rejection. Without an explicit seed it uses OS
cryptographic randomness; seeded requests retain the filter and are labeled
`seeded-practice`. This is not official WCA program approval.

**Native 4×4 companion**:
`turnsmith-wca`, a separate Go executable containing the GPL-3.0-or-later TNoodle
threephase port. The MIT CLI communicates through a bounded process protocol and
does not link the GPL package. Release packages contain both executables, their
licenses, and a separately attached corresponding-source archive. No Java runtime
or network service is required.

**Seeded practice**:
Generation identified by an explicit or daily-derived root seed. It is
reproducible for the recorded method and algorithm version. Predictable practice
seeds do not provide secret competition scrambles. User-owned training setups
are intentional cases rather than uniformly random generated states.
