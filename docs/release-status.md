# Release work and improvement status

Checked 2026-10-30 against the handoff, source, and GitHub ticket discussion.
The handoff contains no separate list of another assistant's suggested
improvements. This is the status of the items it actually identifies; it does
not establish that an unavailable review has been implemented.

| Item | Status and evidence |
| --- | --- |
| v0.1.0 generation, validation, cancellation, extensible registry, deterministic streams, and JSON | Implemented. Existing unit, integration, and golden tests pass. The seed domain and fixtures remain unchanged. |
| Public repository and canonical module | Implemented at `frivas/turnsmith` and `github.com/frivas/turnsmith`. |
| Project and executable naming | Implemented as `turnsmith`, including command entrypoint, help, packages, and documentation. |
| Naming investigation | [Research complete](https://github.com/frivas/turnsmith/issues/2). An existing inventory software product uses Turnsmith; no exclusivity or legal-clearance claim is made. |
| MIT ownership line | Updated to `2026 Francisco Rivas and contributors`, as selected by the maintainer. |
| Go-installed version information | Implemented using embedded module and available VCS metadata, with explicit archive metadata taking precedence. Tested with an actual versioned Go install against an isolated local module proxy; no public tag was created. |
| Windows formatting failure | Reproduced with CRLF checkout conversion and fixed by requiring LF for Go source. Native CI results belong in the [validation ticket](https://github.com/frivas/turnsmith/issues/6). |
| Native CI and final release archives | Partially verified. Local checks and six-target packaging pass; final candidate evidence remains gated by the selected release policy. |
| Signing, notarization, build provenance, and immutable releases | Researched, not implemented or enabled. Await the [release-policy decision](https://github.com/frivas/turnsmith/issues/5). Current archives have checksums. |
| Homebrew tap/formula | Not created. Inclusion or deferral requires the release-policy decision. |
| SemVer tags, changelogs, and multi-platform release automation | Implemented. A valid pushed tag runs CI and automatically publishes six archives, checksums, and a Git-history changelog. Manual previews never publish. |
| Actual v0.1.0 publication | No tag or release created yet. The maintainer selects the candidate and pushes its tag; automatic publication after passing checks was explicitly selected. |

The [release map](https://github.com/frivas/turnsmith/issues/1) remains the canonical
index. Closed ticket resolutions contain decision detail and immutable evidence
links. Do not close a decision ticket merely because its recommended policy has
been researched.
