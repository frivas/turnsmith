# Adding a puzzle

Only 333/random-move ships in v0.1.0. The following describes an extension
workflow; references to 444 are hypothetical and unimplemented.

1. Choose a canonical ID and aliases. IDs are exact strings, not parsed cube
   dimensions. `pyraminx` is as valid an identifier as `444`. Check registry
   collisions and document supported spellings.
2. Describe the puzzle and each actual method. Methods declare a stable ID,
   algorithm version, description, guarantees, limitations, explicit-length
   support, optional default length, and applicable length limits.
3. Implement `puzzle.Generator` in a puzzle-owned package. Accept the context,
   caller-supplied random source, and typed options. Keep state local. Do not
   access global randomness, files, shell commands, output writers, or the
   network. Check cancellation in work loops.
4. Generate using the puzzle's own model and notation. Cube generators may use
   `cube.Move` with a face, layer count, and turn. Size-aware validation rejects
   impossible layer counts. The notation model can express wide moves, but
   that does not establish working larger-cube support. Non-cube generators
   return their own canonical token strings and need not import `cube`.
5. Return a `puzzle.Registration` containing descriptor and method instances,
   then pass it to `puzzle.New` at startup alongside the 333 registration.
   Do not edit CLI parsing, application dispatch, text output, or JSON output
   to recognize a specific new puzzle.
6. Add invariant tests, boundary tests, independent calls, cancellation tests,
   fixed seed vectors, alias tests, discovery metadata checks, and batch-prefix
   checks. Ensure generic CLI/renderers can handle the notation. Existing
   test-only non-cube and non-length fixtures demonstrate those boundaries;
   they are never registered in production.
7. Update documentation and release notes. Algorithms that change existing
   output need a new algorithm version. Do not list a method as supported
   until its actual implementation and tests exist.

The application rejects an explicitly supplied length for a method that does
not support it. A nil length stays omitted from JSON, rather than becoming 25.
Current general safety ceilings remain 1000 exact-length tokens per scramble,
10000 scrambles per batch, and 1000000 total output tokens. Different future
resource models need a deliberate typed validation change, not an unbounded
options map. The registry validates descriptors and clones them for readers;
generators must keep descriptors stable and avoid retaining mutable options.

A real 444 random-move generator needs decisions about layer widths, wide
notation, axes, redundancy, even-cube conventions, and defaults. A random-state
generator needs reachable-state sampling, a solver behind the same boundary,
solution inversion, correctness and performance tests, and licensing review.
Neither extension is merely a different length, and neither automatically
qualifies a tool for official competition.
