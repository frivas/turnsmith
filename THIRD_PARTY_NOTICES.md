# Attribution and notices

The scramble generator and packaging tools were implemented for this project
from its specification. No third-party scrambler, solver, or packaging code was
copied or adapted. There are no external Go module dependencies. No original
scrambler repository or commit was supplied; this project does not assert a
relationship with an existing scrambler.

Deterministic randomness uses Go's standard-library `math/rand/v2.PCG`, based on
Melissa O'Neill's PCG work. See https://www.pcg-random.org/ and Go's implementation
at https://cs.opensource.google/go/go/+/refs/tags/go1.27.3:src/math/rand/v2/pcg.go.
The binary contains the Go runtime and standard library. Their license follows.
The algorithm specification in docs/reproducibility.md records the arithmetic
contract; future adaptations of external implementations must record their
repository, commit, author credit, and license here.

## Cube-state test reference

The facelet state model and terminal renderer are independently implemented.
The checked-in state fixtures were generated using cubejs 1.3.2 as an independent
reference: https://github.com/ldez/cubejs, by akheron and maintained by ldez,
licensed under MIT. Its source code and dependencies are not included in the
product or used by CI. Only the resulting test states are retained.

## Go license

Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

* Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
* Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
