# Catalogue concurrency

Status: Accepted and implemented.

## Context

The catalogue exposes separate whole-document `Read` and `Write` operations.
Both add and remove read a document, modify it, and save it. Locking those
individual calls would still allow concurrent commands to overwrite each
other's changes using stale documents.

## Agreed decisions

- Support Linux and macOS.
- The JSON catalogue implementation owns locking. Callers never acquire or
  release locks themselves.
- Permit concurrent readers and require exclusive access for updates.
- Replace `Write(Document) error` with the atomic read-modify-write operation
  `Update(func(*Document) error) error`, so concurrent changes are not lost.
  Keep `Read() (Document, error)` for reading snapshots. Do not expose a
  whole-document write operation that could save a stale snapshot.
  The JSON implementation holds its exclusive lock across reading the latest
  document, invoking the callback, and saving the result.
- Keep locks limited to catalogue access. Network operations, cloning, and
  checkout deletion run without a catalogue lock.
- Add first reads under a shared lock to check for an existing installation.
  The read releases its lock before returning. If the installation exists,
  return the existing result. Otherwise, clone into a new checkout directory,
  then use an exclusive update to read the latest catalogue and save the
  addition without overwriting other changes.
- Remove selects and removes the entry within an exclusive update, then
  releases the lock before deleting the removed installation's checkout.
  Checkout deletion only follows a successful catalogue update.
- List reads a snapshot under a shared lock. Filtering and formatting the
  returned document do not need to retain the lock.
- Add rechecks for the same repository URL and reference inside the exclusive
  update. If another add has already saved it, keep that first saved entry and
  return the normal successful add payload using its data. Do not distinguish
  this outcome with `already_added`; finding an existing entry in the initial
  read also returns the normal successful payload: `status: "added"`, with
  the saved entry's URL, reference, and checkout path.
- After losing this race, add attempts to delete only its own redundant
  checkout, after releasing the catalogue lock. Ignore cleanup errors without
  reporting them to the user. Later automatic cleanup is expected to handle
  leftovers, but no automatic cleanup implementation exists in the current
  codebase; that is separate work.
- Remove retains its current not-found error when no matching installation
  exists, including when another remove has already removed it.
- When a lock is busy, wait until it can be acquired without a fixed timeout.
- Require shared-reader and exclusive-writer exclusion, without a writer-priority
  or acquisition-order guarantee. Matching Go's `sync.RWMutex` scheduling is
  not a requirement.
- Use `github.com/gofrs/flock` for shared and exclusive file locks. The lock
  must coordinate separate Robert processes and overlapping calls within a
  process, including calls on the same catalogue instance.
- Lock a persistent companion file at the catalogue path plus `.lock`
  (`~/.robert.lock` by default). The existing save operation replaces the JSON
  file through a temporary file and rename, so the lock must have a stable
  identity independent of that replacement.
- Reads may create the lock file, including when the catalogue is absent.
  A missing catalogue still returns defaults without creating the JSON file.
  Releasing a lock does not delete the lock file. Its presence does not mean
  a process currently holds a lock.
- A callback error aborts the update without saving. Release the lock when
  an operation ends, including error paths.
- Preserve atomic file replacement: failed saves leave the previous catalogue
  intact. After a failed save, add attempts checkout cleanup and reports the
  save failure using the existing contextual JSON error behavior. The silent
  cleanup policy above applies specifically to redundant clones after a
  successful duplicate add. Remove never deletes a checkout if its catalogue
  update fails before saving.

## Implementation and verification notes

The update callback receives the latest document after exclusive acquisition.
Add must append to that document, rather than replace it with the snapshot
used before cloning. Duplicate detection remains an add workflow rule; the
catalogue continues trusting saved configuration without new validation.

Network-based default-branch resolution also stays outside catalogue locks.
Remove selects the entry from the latest document while holding the exclusive
lock and retains that entry's exact checkout path for deletion after saving.

Lock ownership must last for each complete operation. The locking library's
thread safety alone does not establish this: repeated acquisition on one
already-locked `Flock` instance can return immediately. Do not let one call
release a lock that another overlapping call still needs.

An error releasing a lock after a successful save must not be treated as a
failed save that triggers deletion of a now-registered checkout. Preserve the
distinction between failing before persistence and failing after persistence
when integrating error handling.

Verify the implementation with both goroutines and independent processes:

- Readers can overlap; updates exclude readers and other updates.
- Concurrent updates to different entries are both preserved.
- Two adds of the same URL and reference keep one saved entry and return its
  data; redundant checkout cleanup occurs after unlocking and errors are ignored.
- Cloning and checkout deletion do not prevent other catalogue operations.
- Concurrent removals preserve the existing not-found behavior, and removal
  deletes only the checkout belonging to its selected entry.
- Callback and persistence failures leave the catalogue unchanged and release
  locks; error output retains actionable storage context.
- Locking remains effective across repeated JSON file replacements and when
  the catalogue does not exist yet.

Run the project suite and race checks, and exercise file-lock behavior on both
Linux and macOS. Cross-compilation alone does not verify macOS lock semantics.
Automatic cleanup of abandoned checkout directories is separate work.

Implementation verification: the full test suite (including independent-process
concurrency tests), race detector, and `go vet` pass on Linux. Production code
cross-compiles for macOS on both amd64 and arm64. macOS runtime lock tests have
not been run in the Linux development workspace.

## References

- [gofrs/flock documentation](https://pkg.go.dev/github.com/gofrs/flock)
- [Go sync.RWMutex semantics](https://pkg.go.dev/sync#RWMutex)
