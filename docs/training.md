# User-owned case training

Training uses collections supplied by the user. No algorithm library is bundled. A collection is JSON:

```json
{
  "schema_version": 1,
  "name": "My practice cases",
  "cases": [
    {"id": "example", "name": "My example", "puzzle": "333", "algorithm": "R U R'"}
  ]
}
```

Case IDs must be unique within the collection. Supported puzzle IDs are `222`, `333` and `444`. Algorithms use the shared parser, including grouping, commutators, conjugates, slices, wide turns and rotations. Each case's setup is the inverse of its algorithm. Collection validation replays setup followed by the algorithm and requires a solved result, including the original orientation. Algorithms must be compatible with the selected cube size.

A collection may contain 1–10,000 cases, up to 10,000 expanded moves per case and 100,000 expanded moves in total. Import size is bounded at 16 MiB. JSON duplicate keys, unknown fields and unsupported schema versions are errors.

# Progress and selection

Progress statuses are `new`, `learning` and `known`. Status changes are explicit. Recording an attempt does not change the status. Each attempt records duration in nanoseconds, whether a hint was revealed, success, timestamp and ID. Identical explicit IDs and contents are idempotent. Conflicting contents under the same case and attempt ID are errors.

Selection supports these policies:

- `balanced` chooses among the cases with the fewest recorded attempts.
- `learning` excludes known cases, then balances attempt counts. It reports no matching cases when all are known.
- `slowest` first chooses cases without a successful unhinted attempt, then those with the greatest mean successful unhinted duration.
- `sequential` follows the most recently recorded case in collection order and wraps around. A new collection begins at its first case.

Ties use the caller's seeded random source in collection order. Selection reads progress without reserving a case. Record the attempt before requesting another case. These policies balance practice, rather than sample cube states uniformly. Hint reveal belongs to the terminal interface; the stored hint flag preserves its effect on statistics.

# Local storage

`training.NewStore(dataDir)` uses `training.json` and `training.json.bak` under the same data directory as timed practice. Its separate `.training.lock` directory serializes training operations across processes. Writes use a synced temporary file, atomic rename, a previous-good backup and directory sync where the platform supports it. Windows retains synced file writes and the previous-good backup without directory sync.

Collection progress is namespaced by SHA-256 of the parsed collection's canonical JSON encoding. Renaming or editing a collection creates a new namespace, so an edited algorithm cannot silently inherit the old timing history. Keep the collection file with exported progress. Progress exports contain the collection fingerprint, case IDs, statuses and attempts, but do not duplicate the algorithms.

Malformed or unsupported stored data causes an error. It never resets progress. Explicit backup recovery validates the backup and preserves existing primary bytes in a uniquely named `training.json.before-recovery-*` file before restoring it. A missing primary with an existing backup also requires explicit recovery. Exports are versioned JSON. Imports either replace explicitly or merge disjoint collections and identical existing collections. Differing progress for the same collection is a conflict requiring an explicit replacement decision.

An abandoned lock is never broken automatically. Inspect `LockInfo`, stop all processes using the store, then call `BreakLock` with the inspected owner token. Cancellation while waiting releases the caller without stealing another process's lock.

Tests cover inverse setups with rotations and wide moves, duplicate and malformed input, all selection policies, status and attempt persistence, deterministic selection, concurrent writers, import conflicts, corruption preservation, backup recovery and lock cancellation. Run `go test -race ./internal/training`.

# Terminal commands

```sh
turnsmith train list --file my-cases.json
turnsmith train --file my-cases.json --mode learning --seed 42
turnsmith train status --file my-cases.json --case example --status known
turnsmith train export --data-dir ./practice-data --output training-backup.json
turnsmith train import --data-dir ./practice-data --input training-backup.json
```

The default action is interactive practice. `--list` and `--json` list validated cases and progress without opening the terminal screen. `--data-dir` selects the local progress directory for every action. Practice requires terminal stdin and stdout; redirected input or output fails before creating progress data. `--mode` chooses `balanced`, `learning`, `slowest` or `sequential`. `--seed` reproduces selection for the same collection and progress. The screen shows the selection seed.

Follow the displayed inverse setup while holding the white face up and green face toward you. Left and right arrows page through setup tokens. Home returns to the first page. Colored tiles show the completed setup. Press Space or Enter to start, then press again to stop. The stopped attempt awaits a correctness decision: `y`, Space or Enter records success; `d` records an unsuccessful attempt. While timing, `d` stops and immediately records failure. Timing uses the monotonic clock rather than wall-clock subtraction.

Press `h` to reveal your algorithm. The same arrow keys page through its canonical moves; the attempt records that a hint was used. `l`, `k` and `0` explicitly mark the current case learning, known or new. `n` selects another case after the current attempt is recorded. Space repeats the current case. `q`, Escape or EOF exits; quitting discards an unfinished attempt. The screen pauses new attempts when too small, while a running or stopped attempt can still be stopped and recorded after a resize.

`train recover` restores the validated previous-good backup. `train lock` prints lock ownership metadata. After stopping every process using the store, `train unlock --token TOKEN` removes the inspected abandoned lock. These actions accept `--data-dir`. Legacy convenience flags `--export-progress` and `--import-progress PATH` correspond to export and import. Imports default to a conflict-checking merge; `--replace` explicitly replaces current progress after validation.
