# Security review, 2026-11-05

| Severity | Findings |
| --- | --- |
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 3 |

No exploitable vulnerabilities were confirmed in the reviewed paths. There
are two dependency advisories outside the imported packages and one group of
18 Staticcheck maintenance diagnostics. These remain open. The security review
did not change dependencies. A subsequent CI reliability fix is recorded below.

## Scope and tools

Reviewed the working tree based on commit
`f3cef2fabe485f33897d1896992696e3936ac27b`, with the README, screenshots, and
capture script added in this update. Automated Go checks covered all 22 root
packages, including both `turnsmith` and `turnsmith-wca`, eight external modules,
and the Go standard library. The credential-pattern scan covered 137 text files
before this report was added, including tracked files and the new documentation
helper. It excluded ignored files and Git history.

Environment: macOS ARM64, Go 1.27.3, Staticcheck 2026.2.2 / 0.8.2, and
govulncheck 1.9.0 using `https://vuln.go.dev`.

| Check | Result |
| --- | --- |
| `govulncheck -show verbose ./...` | Exit 0; no affected symbols or imported packages; two module-only advisories |
| `staticcheck ./...` | Exit 1; 18 diagnostics, detailed below |
| `go vet ./...` | Passed |
| `go test ./...` | Passed |
| `go test -race -short ./...` | Passed |
| `go mod verify` | All modules verified |
| `go run ./scripts/check-format.go` | Passed |
| Credential-pattern scan | No candidates found |

## Dependency findings

### INFO: GO-2026-6629 in an unused package

Confidence: High for the dependency version and scanner result.

Location: `go.mod:16`, `golang.org/x/text v0.31.0`.

The [official advisory](https://pkg.go.dev/vuln/GO-2026-6629) describes a panic
in `golang.org/x/text/secure/precis` when processing crafted input. It is fixed
in v0.41.0. Govulncheck found the vulnerable module version but no imported
affected package or reachable affected symbol in this project. An attack path
through Turnsmith was not established.

Recommendation: upgrade `golang.org/x/text` to v0.41.0 or a later compatible
version in a dependency update, then repeat the full test suite and scan.

### INFO: GO-2026-5970 in an unused package

Confidence: High for the dependency version and scanner result.

Location: `go.mod:16`, `golang.org/x/text v0.31.0`.

The [official advisory](https://pkg.go.dev/vuln/GO-2026-5970) describes an
infinite loop in `golang.org/x/text/unicode/norm` on invalid UTF-8 input. It is
fixed in v0.39.0. Govulncheck found no imported affected package or reachable
affected symbol. An attack path through Turnsmith was not established.

Recommendation: the same upgrade to at least v0.41.0 addresses both advisories.

## Static analysis finding

### INFO: 18 maintenance diagnostics

Confidence: High for the diagnostics; no security impact was confirmed.

| Rule | Count | Locations and meaning |
| --- | --- | --- |
| SA4006 | 2 | `internal/cli/train.go:288`, `internal/randomstate/memory_test.go:10`; assigned values are not used |
| SA1019 | 6 | `internal/release/source.go:130`, `internal/terminal/paint.go:53`, and `internal/terminal/terminal_test.go:208,213,217,224`; deprecated tar constant or terminal cell access API |
| U1000 | 10 | Unused solver helpers in `internal/tnoodle444/centercube.go:30`, `cornercube.go:27`, `edgecube.go:38`, `fullcube.go:55,117`, `moves.go:208,227`, and `util.go:74,82,83` |

These diagnostics cause Staticcheck to fail. They are not a clean Staticcheck
result and should not be presented as one. Review the solver helpers against
the upstream port before removing them. The deprecated tar constant is used
to accept old regular-file entries; evaluate compatibility before changing it.

Recommendation: address these diagnostics in a separate maintenance change,
with terminal and solver regression tests.

## Input and process review

The manual review focused on the boundaries exposed by this local CLI:

- Imported sequences pass through bounded reads, a puzzle-specific move parser,
  and cube-state validation. The algorithm parser limits input size, expanded
  moves, and group nesting. See `internal/cli/sequence.go`,
  `internal/cli/commands.go`, and `internal/alg/alg.go`.
- Training collections limit bytes, case counts, and expanded moves. Names and
  IDs reject control characters. Session imports validate IDs, text, timestamps,
  and durations, with size limits and strict JSON validation. See
  `internal/training/collection.go` and `internal/practice`.
- Interactive painting accepts color styling and discards other escape
  sequences instead of executing cursor or OSC commands from frame text.
  See `internal/terminal/paint.go`.
- The 4×4 companion launches directly with `exec.Command(path, "--stdio")`,
  without a shell. The executable path comes from installation lookup or an
  explicit environment override, not imported algorithms. Protocol reads,
  diagnostic output, random requests, and move counts are bounded; moves are
  parsed before use. Cancellation terminates the process. See
  `internal/wcaengine/client.go`.
- Practice writes use private temporary files, atomic replacement, backups,
  and locks. User-selected file destinations are intentional CLI inputs.
  Release source extraction rejects traversal paths, symlinks, and special
  entries. See `internal/practice/store.go`, `internal/training/store.go`, and
  `internal/release/source.go`.
- Unseeded WCA generation uses `crypto/rand.Reader` and rejects entropy failure.
  Deterministic practice seeds are explicit. No production network client,
  insecure TLS override, weak cryptographic primitive, or shell execution sink
  was found by the targeted searches.
- The screenshot helper uses argument-list subprocess calls and temporary
  practice storage. Its optional Python dependencies are documentation tools,
  separate from the shipped Go applications; they were not covered by the Go
  vulnerability scan.

## CI reliability follow-up

The first macOS CI attempt on the documentation commit failed
`TestNativeRejectsMalformedResponses/child-failure` because the child's final
stderr text was missing from the returned error. The same attempt passed on
Windows and Linux, and the macOS retry passed. The test exposed a race in error
reporting: stdout EOF can arrive before Go's stderr copier finishes.

The follow-up fix calls the existing process shutdown and wait routine before
snapshotting stderr on a failed protocol read. A new regression test holds
stderr delivery behind a channel. It fails on the previous code, then passes
with the fix. The test retains the real subprocess and Go's stderr copier.

Validation of the fix passed the complete uncached Go suite, the short race
suite, `go vet ./...`, formatting, Staticcheck on `./internal/wcaengine`, and
20 repetitions of the complete companion-process test suite. The 18 existing
project-wide Staticcheck diagnostics are separate from this fix.

## Secrets scan and limits

No credential candidates were found using patterns for provider tokens,
private keys, credential-bearing database URLs, literal credential assignments,
or tracked credential filenames. This was a pattern scan, not a complete
entropy-based audit or a Git-history credential audit.

This is a point-in-time check with targeted manual review, not a guarantee
that the application has no vulnerabilities. Native tests ran on macOS;
Windows and Linux behavior depends on the repository's cross-platform CI.
The installed binaries and previously published release archives were not
rescanned or rebuilt as part of this documentation update.
