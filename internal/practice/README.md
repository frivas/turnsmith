# Optional local practice storage

The CLI must construct and use this store only for an explicitly invoked practice
command. `New` resolves the configured path but creates no files. With no override,
the location is `os.UserConfigDir()/turnsmith`; callers can provide `--data-dir`.

`Sessions`, `Get`, `Create`, `Record`, `Update`, `Delete`, and `DeleteSession` use a
single versioned `sessions.json`. `Record` accepts already-measured duration, not
wall-clock start/stop values. The timer caller should use Go's monotonic clock and
call `Record` after stopping, rather than holding any store lock during the solve.
CreatedAt is supplied by the caller or defaults to the recording time in UTC.

## Data and summaries

Schema version 1 contains named sessions and ordered solves with stable IDs,
creation timestamps, exact nanosecond durations, penalties, scramble text,
generation metadata, and notes. `DurationNS` is an int64 in Go; JSON encodes it as
a decimal string to avoid floating-point precision loss in other consumers.
CSV likewise uses decimal integer nanoseconds. Timestamps preserve nanosecond
precision and explicit offsets. `Record` normalizes its supplied timestamp to UTC.

`Summarize` expects validated solves, as returned by the store. Current ao5/ao12
use the last 5/12 records in stored order, discard one fastest and one slowest,
and average the remaining adjusted times. One DNF is removable as the worst;
two make the average DNF. +2 adds two seconds before ranking and averaging.
Best/worst singles exclude DNFs when valid singles exist; DNFCount is separate.
Insufficient windows have Available=false. All-DNF singles have DNF=true.
Average nanoseconds truncate only the final sub-nanosecond remainder. These are
local practice measurements, not a claim of official competition measurement.

Sources: [WCA ao5 and DNF rules](https://www.worldcubeassociation.org/regulations/#9f8)
and [csTimer's default 5% rounded-up trimming](https://github.com/cs0x7f/cstimer/blob/master/src/js/stats/timestat.js).
For windows 5 and 12, this trims one result from each end. No source code from
these projects is used here.

## Writes, locks and recovery

Each operation takes an exclusive `.practice.lock` directory with a random token,
owner PID and creation timestamp. Waiting is bounded to five seconds or the
caller's context deadline. Reads use the same lock, so no process observes a
half-committed application operation. Each mutation reloads under that lock,
preventing lost updates from other CLI processes. Nothing is cached globally.

Writes use a same-directory private temporary file, flush it, and rename it over
the target. A previous validated snapshot is written to `sessions.json.bak`
before replacing the primary. Unix directory entries are fsynced as well; Go's
Windows interface does not expose directory fsync. Temporary files are cleaned
on normal failure paths. Created directories request 0700 and files request 0600;
Windows protection also depends on the containing directory's ACLs.

Malformed data is an error, never an empty store. A missing primary with an
existing backup is also an error. `Recover` explicitly restores a validated
backup without replacing it with corrupt primary data. It restores the *prior*
snapshot, so the most recent mutation may be absent. Invalid backups stay intact
for inspection and are not silently used.

Locks are never broken because they are old, or merely because a PID looks
reusable. After a crash, the user must stop every process using this data directory
and inspect the lock. `LockInfo` identifies its token/owner. Only after confirming
that it is abandoned may an administrative command call `BreakLock` with the
inspected token. An incomplete lock with no owner metadata requires manual
inspection/removal after all writers are stopped. Do not expose an automatic
live-lock deletion retry.

## Import/export

`ExportJSON` and `ExportCSV` take a consistent snapshot, then release the lock
before writing to the caller. Imports parse and validate the entire input before
performing one locked mutation. Unknown schema versions/fields, duplicate IDs or
JSON keys, invalid types, durations, timestamps, penalties, control characters,
and mismatched CSV metadata are rejected without changing stored data.

Merge is the default: matching solve IDs with identical content are skipped,
so repeated imports are idempotent. Different content under an existing ID or a
name owned by another session is an error. Replace must be explicitly selected;
it replaces the complete document and retains the previous snapshot as backup.
An already corrupt primary must be explicitly recovered before further imports.

CSV version 1 includes a `kind=session` row for every session, including empty
sessions, followed by `kind=solve` rows. It is Turnsmith's own interchange format,
not a csTimer-compatible format. Preserve its header, IDs and timestamp fields.
Notes may include LF newlines/tabs but no other control characters; CSV quoting
preserves commas, quotes and those newlines. This is plain data, not a spreadsheet
formula format.

The v1 store is limited to 64 MiB, 1000 sessions and 100000 solves. Notes are
limited to 16 KiB, scramble strings to 100000 bytes. Imports/reads are bounded,
but a generic caller-supplied Reader or Writer cannot be forcibly interrupted by
context if its own Read/Write blocks. File recovery and cross-process operations
are tested separately from memory-only statistics.
