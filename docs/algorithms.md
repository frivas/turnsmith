# Algorithms

## 333 / random-move / algorithm version 1

Default length is 25 emitted tokens. Allowed lengths are 1 through 1000. A
batch contains 1 through 10000 scrambles, limited to 1000000 total tokens.
The application checks `length > 1000000 / count` after checking the count
bounds, so it never needs an overflowing multiplication.

Face enumeration is always `U, D, L, R, F, B`. Opposite pairs are `U/D`, `L/R`,
and `F/B`. Turn enumeration is clockwise quarter, half, counterclockwise quarter,
rendered as no suffix, `2`, and ASCII `'`. Every suffix counts as a single move.

For each token, enumerate the six faces once into a fixed-size candidate array:

1. Exclude the preceding face, if any.
2. If the last two faces are A followed by opposite(A), exclude A.
3. Choose an unbiased integer below the remaining candidate count and take
   that face in enumeration order.
4. Independently choose an unbiased integer below 3 and use that turn amount.
5. Append a typed one-layer move, then render all moves to canonical tokens at
   the generator boundary.

There are six candidates initially, five after one move, and four or five
thereafter. Faces are never proposed and rejected repeatedly. Candidate selection
examines exactly six faces per token and uses constant-size recent history.
The bounded-integer primitive discards raw random words in an incomplete residue
interval to remove modulo bias. This very rare numeric rejection is distinct
from rejecting moves; as with exact unbiased bounded-integer sampling generally,
it has no fixed upper bound on random-word draws.

No adjacent same-face move is permitted, including differing turns. No
A-opposite(A)-A window is permitted, since commuting the middle move permits
combining the repeated faces. The algorithm does not prohibit every possible
longer redundancy or guarantee that different scrambles reach different states.
There is no simulator or visited-state tracking.

Each valid face and turn is selected uniformly at that step. This does not
uniformly sample reachable states. A sequence can return to a prior state, and
this method is not approved for official competition. Arbitrary legal outer-face
moves need no solved-state representation to establish their legality.

## Future methods

No random-state method is implemented. It would need a reachable-state model,
documented state sampling, a reviewed solver or dependency, solution inversion,
tests applying generated scrambles to sampled states, performance checks, and
license review. Even those steps alone do not establish competition compliance.

Larger cubes need size-specific layer widths, notation, axis restrictions,
length defaults, redundancy rules, and even-cube conventions. Changing the
length does not implement another cube size.
