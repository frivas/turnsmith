# Native 4×4 random-state backend

Research checked 2026-10-31. The user rejected a Java dependency and asked for a
native Go port. No solver code was copied into Turnsmith and no Java runtime was
installed during this investigation.

A Go port of TNoodle's 4×4 engine is technically feasible. The core reduction
engine is about 3,200 Java lines. Its license is GPL-3.0-or-later, including the
solver component itself. Putting the port in an MIT package would not preserve
that license. The immediate decision is how to distribute the GPL engine, then
how to verify the port.

## What the WCA requirements mean

Current Regulation 4b3 requires uniform random states with a minimum solving
distance of two moves for 4×4. Its random-move exception lists 5×5, 6×6, 7×7, and
Megaminx. 4×4 is absent. Regulation 4b3b raises the 2×2 minimum to four moves.
Turnsmith's existing 4×4 random-move method cannot become conforming by increasing
its length. Regulation 4b separately requires the current official scramble
program for official competitions, so even a verified Go port should describe
its output as competition-style practice rather than claim official approval.
[WCA regulations, Article 4](https://www.worldcubeassociation.org/regulations/#4b3)

## Exact source and license

The inspected TNoodle-lib snapshot is
`d01a947d9f028f38085cda9b0507a9cf3d3f38a8`. The official server is a separate
repository. Its AGPL license is not the license being inferred for this port.
The library root carries GPLv3; the threephase Search header and README explicitly
grant GPL version 3 or later, crediting Shuang Chen, 2014. The min2phase Search
header likewise grants GPL version 3 or later, crediting Shuang Chen, 2015.
These headers settle the component question without relying on the server's
license. [Library license](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/LICENSE),
[threephase Search header](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/Search.java),
[threephase README and license](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/README.md),
[min2phase Search header](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/min2phase/src/main/java/cs/min2phase/Search.java)

A translated port remains a derivative. Retain attribution, GPL text, modification
notices, corresponding source, and build instructions with its distribution.
A standalone GPL command with an ordinary text/JSON interface is an option for
keeping independently licensed MIT source separate. A process boundary alone
does not establish that two programs are independent; do not present it as an
automatic exemption. A linked executable containing the GPL port needs a
GPL-compatible distribution policy. Existing MIT source can retain its MIT
notices. [GPLv3, sections 4–6](https://www.gnu.org/licenses/gpl-3.0.html)

The source archive was inspected in `/tmp/turnsmith-tnoodlelib-audit/` only. Counts
below include comments and blank lines, measured from the pinned snapshot.

| Part | Java files | Lines | Bytes | Role |
| --- | ---: | ---: | ---: | --- |
| threephase | 12 | 3,159 | 70,798 | 4×4 random-state sampling and reduction |
| min2phase | 6 | 2,508 | 73,985 | Final reduced 3×3 solve |

The 12 threephase files are `Center1`, `Center2`, `Center3`, `CenterCube`,
`CornerCube`, `Edge3`, `EdgeCube`, `FullCube`, `Moves`, `Search`, `Tools`, and
`Util`. [Pinned source directory](https://github.com/thewca/tnoodle-lib/tree/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase)

## Port structure

The entry point is `FourByFourCubePuzzle.generateRandomMoves`. It calls
`Search.randomState`, then canonicalizes the resulting notation through
`AlgorithmBuilder`. `Search.randomState` constructs a random `FullCube` and runs
the staged search. The inherited `Puzzle.generateWcaScramble` repeats generation
when `solveIn(wcaMinScrambleDistance - 1)` finds a short solution. Preserve that
filter explicitly; a long printed algorithm does not prove the sampled state is
far enough from solved.
[FourByFourCubePuzzle](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/scrambles/src/main/java/org/worldcubeassociation/tnoodle/puzzle/FourByFourCubePuzzle.java),
[Puzzle generation and distance filtering](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/scrambles/src/main/java/org/worldcubeassociation/tnoodle/scrambles/Puzzle.java)

A practical division of work follows the source's dependencies:

1. Port cubie representation, coordinate ranking, moves, symmetry operations,
   and random-state sampling. Compare cubie moves against Turnsmith's independent
   4×4 facelet simulator before building the search.
2. Port Center1/2/3 tables and Edge3 symmetry/pruning tables. Check table dimensions,
   sentinel values, representative counts, and deterministic checksums.
3. Port the staged Search and its priority ordering. Preserve its candidate
   limits and stable tie behavior before attempting speed changes.
4. Connect the reduced 3×3 solve. The dependency is narrow: Search creates a
   54-character URFDLB facelet string, asks min2phase for its solution, parses the
   moves, and appends them. An adapter to the existing native Go 3×3 solver may
   avoid translating another 2,508 lines, provided orientation, move directions,
   legality, and complete solving are tested. This is an implementation inference,
   not a claim of byte-identical TNoodle output.
5. Add cancellation, one-time table construction, isolation of per-search mutable
   state, output normalization, the minimum-distance rejection, and versioned
   randomness metadata. Do not silently substitute random moves on failure.

The source samples centers by shuffling 24 colored entries, wings by shuffling
24 distinct entries, and corners through uniformly sampled permutation/twist
coordinates. Port the bounded RNG without modulo bias. Java-seed compatibility
requires Java Random's exact semantics; deterministic Go seeds can instead define
a new documented algorithm version. Either approach needs target-state replay
checks independent of the solver.
[CenterCube](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/CenterCube.java),
[EdgeCube](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/EdgeCube.java),
[CornerCube](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/CornerCube.java)

## Tables and translation hazards

The largest persistent threephase allocation is Edge3's pruning table. Its
31,006,080 entries pack two bits each into 1,937,880 Java ints, or 7,751,520 raw
bytes. `MAX_DEPTH` is 10. Preserve the modulo-three pruning encoding and its
sentinels rather than interpreting it as a normal byte-per-distance table.
Center1 uses a 15,582 × 36 int move table and a temporary 735,471-int reverse
mapping. Center2 uses 6,435-entry center coordinates; Center3 has
35 × 35 × 12 × 2 coordinates. Use fixed-width Go storage for packed tables.
[Edge3](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/Edge3.java),
[Center1](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/Center1.java),
[Center2](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/Center2.java),
[Center3](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/Center3.java)

Java `int` arithmetic is 32-bit, `byte` is signed, `char` is unsigned 16-bit,
and `>>>` shifts logically. Go `int` on the release targets is 64-bit, so literal
translations can corrupt packed permutations or pruning values. Java initializes
static arrays in declaration order; make dependencies explicit in Go. Replace
PriorityQueue with a tested heap and preserve comparator direction. Search owns
mutable buffers, so sharing one Search among goroutines would require protection.

The upstream README reports historical Java results on an i7-2670QM of about
250 ms per solve, 6–7 seconds initialization, and at most 30 MB including
min2phase. These are old upstream observations, not measurements or promises for
Go. The optional serialized table is about 20 MB according to that README;
first ship computed tables with cancellation or version/checksum any persisted
cache. [Upstream performance and cache notes](https://github.com/thewca/tnoodle-lib/blob/d01a947d9f028f38085cda9b0507a9cf3d3f38a8/threephase/src/main/java/cs/threephase/README.md)

## Other backends checked

`tnoodle-rs` 0.1.1 is a Rust port with the threephase engine and Java differential
fixtures. Its own documentation explicitly states GPL-3.0-or-later, so it does
not solve the licensing issue. It was released October 7, 2026. Its documented
3×3 search timing differs from Java by default, and some table serialization is
omitted. The published crate excludes the Java fixtures. No six-target standalone
scramble executable was verified in this investigation. It is a useful second
reference for a Go port, rather than evidence of official approval.
[Crate source, tests, license, and deviations](https://docs.rs/crate/tnoodle-rs/latest)

`cubing.rs` documents alg, kpuzzle, and puzzle APIs; its README does not establish
a complete 4×4 scramble solver. Do not confuse it with `tnoodle-rs`.
[cubing.rs README](https://github.com/cubing/cubing.rs)

`cubing.js` has a real 4×4 random-state implementation. The inspected snapshot
`5133dc49698e9911466cebb8bb1826c73fd4e5ba` contains a roughly 69 KB TypeScript
threephase adaptation, a small math helper, and a separate random 3×3 stage.
Its README describes original code as MPL/GPL dual licensed and vendored parts
as MIT/Apache. However, the license file beside this particular vendor tree
still contains GPLv3, and the solver has no per-file MIT grant. The `vendor/mit`
pathname alone is insufficient to settle the exact port's license. Verify the
specific grant and all adapted contributions before choosing a permissive port
from it. It is not currently a verified permissive shortcut for Turnsmith.
[Project licensing](https://github.com/cubing/cubing.js/blob/5133dc49698e9911466cebb8bb1826c73fd4e5ba/README.md#license),
[4×4 solver](https://github.com/cubing/cubing.js/blob/5133dc49698e9911466cebb8bb1826c73fd4e5ba/src/cubing/vendor/mit/cs0x7f/cstimer/src/js/scramble/444-solver.ts),
[Adjacent license](https://github.com/cubing/cubing.js/blob/5133dc49698e9911466cebb8bb1826c73fd4e5ba/src/cubing/vendor/mit/cs0x7f/cstimer/LICENSE)

No ready, permissively licensed, pure-Go 4×4 random-state engine was verified.
That is a search result, not proof that none exists. A native Go TNoodle port is
substantial but bounded work; it should not be called complete until its states,
move conventions, near-solved rejection, table generation, cancellation, seed
reproducibility, and all six platform builds have passed independent checks.

## Port execution note

The chosen implementation keeps a separate native GPL-3.0-or-later backend
executable and preserves Turnsmith's existing source notices. The Center1,
Center2, and Center3 translation uses checked context-aware table construction,
explicit uint32 occupancy bitsets, and nonnegative int coordinate indices whose
maximum is below 2^31. Table construction finishes before searches read the
shared tables. The temporary Center1 reverse map is discarded afterward.

The center tests exhaust all 735,471 first-phase raw coordinates, all 70
second-phase side/parity coordinates, all 6,435 second-phase center coordinates,
and all 29,400 third-phase coordinates. They also check the 15,582 symmetry
representatives under all 48 symmetries, symmetry inverses and move permutations,
reconstruction after dropping the reverse map, pruning-table reachability and
shortest-path gradients, and cancellation before construction. Center moves
agree with the separate cubie representation for all 36 standard/wide moves.
These checks cover the translated center modules; they do not yet establish the
whole scramble pipeline or official WCA approval.
