# WCA-distance-filtered practice

Researched on 2026-10-31 against the [April 1, 2026 WCA Regulations](https://www.worldcubeassociation.org/regulations/#4b3). Regulation 4b3 requires uniform random states at least two moves from solved; 4b3b raises the 2×2 minimum to four. The random-move exceptions in 4b3e cover 5×5, 6×6, 7×7 and Megaminx. They do not include 4×4. Blindfolded events additionally require random orientation under 4b3a, which this standard-event implementation does not provide.

Official competitions must use the current official scramble program. The WCA [scrambles page](https://www.worldcubeassociation.org/regulations/scrambles/) currently specifies TNoodle-WCA-1.2.3 and says it performs the regulation's filtering. Turnsmith's native methods are practice generators; their method name does not establish official approval.

## Native implementation

`Cube333WCA` and `Cube222WCA` expose method `wca`, algorithm version `1`. Both use the existing reachable-state samplers and native solver tables. They reject a target only when it fails the minimum-distance condition, before solving. Every retained target has the same acceptance condition, so rejection preserves the uniform conditional distribution when the caller supplies independent uniform random words.

For 333, all states at distance less than two are solved or one of eighteen outer-face turns from solved. Half turns count as one move. The filter compares full cubie states against exactly these nineteen states. It does not mistake a long nonoptimal solver sequence for a distance proof.

For 222, distance is read from the complete 3,674,160-state pocket table modulo whole-cube rotation. Distances zero through three exclude exactly 385 classes. An independent test searches all eighteen face turns using the geometric facelet simulator and canonicalizes each state under all twenty-four rotations. Its excluded set matches the table's set exactly. This checks that restricting the solver to U/F/R turns with a fixed corner does not overestimate the relevant shallow distance.

Sampling permits at most 1,024 rejected targets. Exhaustion returns an error. Context cancellation is checked before each random draw, while waiting for tables and during search. Once an admissible target is chosen, a solver failure returns an error without sampling another target. The solver verifies its solution against that target, then returns its inverse. Tests separately replay emitted sequences through the geometric simulator and compare all applicable stickers with the sampled target.

## Entropy and reproducibility

The `DirectEntropy` capability asks the application to supply a caller-owned entropy stream for unseeded generation. A finite 64-bit root seed cannot represent every 333 state: the reachable state space has more than `2^64` elements. Consequently a PCG stream derived from a 64-bit root must not be described as literal uniform selection over the full state space.

An explicit seed remains useful for reproducible practice and has separate result metadata. Its sequences are deterministic, rather than a claim that choosing a uniform 64-bit seed covers all cube states uniformly. Versioned golden sequences pin that mode. Legacy `random-state` version 1 and 333 `random-move` streams remain unchanged.

The CLI-native 2×2/3×3 methods do not implement 4×4 random-state generation, blindfolded orientation, official program approval or competition secrecy. Those need separate implementation or the official tool. Bounded search can fail on an accepted target; the application must propagate that failure rather than silently retry it.

## Verification

Scripted random sources force a solved target followed by an admissible target, proving that shallow states are rejected before solving. Another source repeatedly supplies solved targets to verify the hard rejection bound. Tests cover cancellation, unsupported fixed length, exhaustive 333 depth-one membership, the independently reconstructed 222 shallow set, accepted-target replay and deterministic golden sequences. Existing legacy golden tests continue to pass.


## Native 4x4 extension

The separate GPL-licensed `turnsmith-wca` executable now ports TNoodle's 4x4
three-phase engine to Go. It samples corner coordinates, center colors and wing
permutations, rejects shallow states modulo whole-cube rotation, reduces the
sample and uses the native 3x3 callback. Every returned scramble is checked against
its sampled target using the independent geometric model. One hundred fixed
sampled targets, exhaustive shallow classes, transition/pruning invariants,
cancellation and resource bounds are covered. The main MIT CLI communicates over
a bounded JSON-line entropy protocol and does not link the GPL package.

First initialization plus a target took about 0.56 seconds on this Mac; a warm
three-seed benchmark measured 54 milliseconds per target. These observations are
not latency guarantees. Backend source, licenses and corresponding-source
archives remain separate from any claim of official competition approval.
