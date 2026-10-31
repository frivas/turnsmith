# Release wayfinding

The current execution map is [Deliver the complete terminal practice roadmap](https://github.com/frivas/turnsmith/issues/8).
The historical first-release map is [Map the route to Turnsmith's first public release](https://github.com/frivas/turnsmith/issues/1).
Decisions and their evidence live in GitHub Issues. This document explains how
to continue the map; it does not duplicate ticket answers.

The project and executable name is `turnsmith`, owned on GitHub by `frivas`.
The canonical Go module is `github.com/frivas/turnsmith`, with entrypoint
`cmd/turnsmith`. The user confirmed the executable rename after charting the map.
The user also selected Francisco Rivas and contributors for MIT attribution
and embedded Go metadata for versioned Go installations. The identity ticket
records the resolution and verification of those choices. The map covers the first public release, with existing
v0.1.0 generation behavior retained.

## Tracker operations

Use the GitHub tracker conventions in the Wayfinder skill's companion
`setup-matt-pocock-skills/issue-tracker-github.md` when installed.

- Map label: `wayfinder:map`.
- Ticket labels: `wayfinder:research`, `wayfinder:grilling`, `wayfinder:task`.
- Children: native GitHub sub-issues of the map.
- Blocking: native GitHub issue dependencies.
- Claim: assign the ticket to `frivas` before working on it.
- Resolve: add a resolution comment, close the ticket, then add a titled link
  and a one-line gist to the map's Decisions so far section.
- Human decisions require the maintainer's answer. Charting does not settle them.

Query the frontier by listing the map's native children and selecting open,
unassigned tickets with no open blockers, preserving the returned order:

```sh
gh api --paginate repos/frivas/turnsmith/issues/1/sub_issues \
  --jq '.[] | select(.state == "open" and (.assignees | length) == 0 and (.issue_dependencies_summary.blocked_by // 0) == 0) | {title, html_url}'
```

GitHub repository creation must use the `frivas` account, followed immediately
by switching the active account back to `frivas-at-navteca`. For subsequent issue
maintenance or pushes, use a process-scoped `frivas` credential without changing
the default account. Never print or save the credential in repository files.

Research evidence is maintained on `research/turnsmith-name` and
`research/release-distribution`, with file/commit links in the ticket resolutions.
Use the resolution links for the exact evidence snapshot. Research findings do
not settle command naming, distribution costs, signing policy, or publication
approval on the maintainer's behalf.

## First-release completion

The maintainer requested delivery of the first visible release and reconciliation
of completed work. The current first-release scope is GitHub archives and Go
installation, with automatic SemVer publication, changelogs, and checksums. Paid
platform signing, notarization, provenance, and Homebrew are follow-up work.
The map's release tickets should close with source, CI, package, and publication
evidence once their concrete work is complete. Historical chart-only rules do
not require retaining completed execution tickets after that scope expansion.
