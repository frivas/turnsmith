# Timed sessions and local history

```sh
turnsmith timer --session practice --inspection 15
turnsmith sessions show practice
turnsmith sessions show practice --json
turnsmith sessions add practice --time 12.345 --penalty +2 --notes "missed AUF"
turnsmith sessions edit practice --id SOLVE_ID --penalty DNF
turnsmith sessions delete practice --id SOLVE_ID
```

The timer uses Space/Enter presses with a monotonic clock. With inspection enabled,
the first press starts inspection, the second starts timing and the third stops.
Inspection beyond 15 seconds adds +2; beyond 17 seconds records DNF. This is local
practice timing, without an official-competition measurement claim. Starting an
attempt needs enough screen space to show its full scramble. Resizing never
prevents stopping a running attempt. q discards a running attempt; stopped solves
are already saved. n prepares another scramble. After saving, 2 sets +2, d sets
DNF and 0 clears the penalty. Edit notes afterward using sessions edit.

Sessions use best/worst valid singles and the latest ao5/ao12. Averages drop one
fastest and one slowest adjusted result. +2 adds two seconds; one DNF can be
removed as the worst, two make the average DNF. Insufficient windows display a
dash. Stored order determines the latest window.

## Storage and backup

Only explicit session/timer/training commands access practice files. Generation
and inspection save nothing. All session commands accept `--data-dir PATH`.
By default data is under the OS configuration directory:

- macOS: `~/Library/Application Support/turnsmith`
- Linux: `$XDG_CONFIG_HOME/turnsmith`, or `~/.config/turnsmith`
- Windows: `%AppData%\turnsmith`

```sh
turnsmith sessions export --format json --file backup.json
turnsmith sessions export --format csv --file times.csv
turnsmith sessions import --format json --file backup.json
turnsmith sessions import --format csv --file times.csv --replace
```

Exports include every session. Import merges by IDs by default; repeated identical
imports are idempotent. Conflicting IDs or names fail. `--replace` replaces all
sessions, keeping the previous valid snapshot as backup. JSON and CSV are
Turnsmith schema version 1 formats, including precise decimal nanosecond durations,
IDs, timestamps, penalties, scrambles, metadata and notes. CSV also preserves empty
sessions and multiline notes. Other apps' exports require conversion.

Writes replace `sessions.json` atomically from a synced temporary file. The
previous validated snapshot is `sessions.json.bak`. Corrupt files cause errors
and are never silently reset. `turnsmith sessions recover` restores a validated
backup, which may omit the most recent operation. Keep regular exported backups.
The store limit is 64 MiB, 1000 sessions and 100000 solves.

Operations serialize through `.practice.lock`. A crashed owner's lock is not
removed automatically. Stop every process using the directory, inspect
`turnsmith sessions lock`, then use `turnsmith sessions unlock --token TOKEN`
with that exact token. An incomplete lock without owner metadata needs manual
inspection after all writers are stopped.
