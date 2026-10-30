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
| Native CI and release archives | Native macOS/Linux/Windows CI and six-target packaging passed the hosted preview. The final `v0.1.0` tag pipeline and asset verification are recorded in the [validation ticket](https://github.com/frivas/turnsmith/issues/6). |
| Signing, notarization, build provenance, and immutable releases | Deferred beyond this release. The current distribution uses checksums and documents unsigned macOS/Windows binaries; these optional additions are not claimed as implemented. |
| Homebrew tap/formula | Deferred beyond v0.1.0; no formula or installation claim is made. |
| SemVer tags, changelogs, and multi-platform release automation | Implemented. A valid pushed tag runs CI and automatically publishes six archives, checksums, and a Git-history changelog. Manual previews never publish. |
| Actual v0.1.0 publication | The `v0.1.0` tag selects commit `2017083abfd8ee5ce444631ba6c3ca304b918ca1`. See the [release](https://github.com/frivas/turnsmith/releases/tag/v0.1.0) and [tag workflow](https://github.com/frivas/turnsmith/actions/runs/41095533734) for publication status. |

The [release map](https://github.com/frivas/turnsmith/issues/1) remains the canonical
index. Closed ticket resolutions contain decision detail and immutable evidence
links. The current release scope is GitHub archives and Go installation, with changelogs
and checksums. Signing, provenance, and Homebrew are follow-up work, not unfinished
implementation claimed by the release tickets. The [root changelog](../CHANGELOG.md)
is visible independently of release artifacts.
