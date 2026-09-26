# Automated CLI QA

Run from the repository root with Go, Git, and Python 3.9+ installed:

```sh
python3 scripts/qa_cli.py --workers 16 --rounds 5 --report /tmp/robert-qa-results.json
```

The harness builds the current CLI and launches independent processes against
real, tiny local Git repositories. Git URL rewriting keeps the test offline.
Each scenario uses a temporary child-process home, catalogue, and installation
directory. The real user's configuration and repositories are untouched.
The temporary root, build, source repositories, catalogues, lock files, and
checkouts are removed on completion, including when a scenario fails. The JSON
report records whether removal of the temporary root was verified.

Stdout contains a JSON summary; stderr shows scenario progress. The report also
contains every CLI invocation, exit code, stdout, stderr, and elapsed time.
Exit codes are `0` for no findings, `1` for a failed assertion/setup/cleanup,
and `2` for reproduced product issues. Permission probes require a non-root
Linux or macOS user; root runs explicitly mark them skipped. This run was on
Linux, so it does not establish macOS behavior.

## Coverage

- Default branch, explicit branch, tag, and full commit additions; repeated adds;
  ambiguous removals; exact-reference removals; empty and missing catalogues.
- Invalid arguments, unavailable repositories/references, lock failures, and
  permission-induced catalogue save failures, including rollback and cleanup.
- Five rounds each of 16 distinct concurrent adds/removes, 16 duplicate
  adds/removes, 48 mixed operations (16 adds, 16 removes, 16 lists), and 32
  operations racing to add/remove the same installation.
- JSON shape and exit status, saved entries versus list output, unique entries,
  checkout paths, exact commit IDs, worktree contents, clean working trees,
  shallow history, branch tracking, detached tag/commit checkouts, and absence
  of orphan checkouts or temporary catalogue files after ordinary operations.
- Actual filesystem permission failures during removal and redundant-clone
  cleanup. The latter uses a Git wrapper to synchronize two real clones before
  catalogue publication; the wrapper does not mock Git operations.

## Result: 2026-09-26

Tested application revision: `71aa25d75df2dbbcf3288e02e473bab39f421459`.
See [the recorded summary](qa/cli-2026-09-26.json) for platform details, scenario
results, timings, and findings. The full invocation log from this run is at
`/tmp/robert-qa-results.json`; rerunning the command replaces that file.

**1,057 CLI invocations; 25 passing scenarios; two scenarios reproduced cleanup
issues.** Normal concurrent operations preserved entries and removed their
checkouts. No assertion failures, invalid JSON, lost updates, or deadlocks were
observed. Total harness wall time was 3.962 seconds, including a 149 ms build,
fixture setup, verification, and cleanup.

Sequential timings include CLI process startup and local Git operations, but
exclude the separate verification commands. Samples use filesystem caches;
these are not internet-clone or large-repository benchmarks.

| Operation | Samples | Median | p95 | Maximum |
| --- | ---: | ---: | ---: | ---: |
| Add, explicit main branch | 20 | 17.99 ms | 19.05 ms | 21.55 ms |
| Remove | 20 | 1.71 ms | 2.02 ms | 2.45 ms |
| List | 40 | 1.34 ms | 1.64 ms | 2.13 ms |

Median batch wall times over five rounds, including thread scheduling and
process startup, excluding subsequent state verification:

| Concurrent workload | Commands per batch | Median batch wall time |
| --- | ---: | ---: |
| Distinct adds | 16 | 41.73 ms |
| Distinct removes | 16 | 7.24 ms |
| Duplicate adds | 16 | 37.91 ms |
| Duplicate removes | 16 | 7.43 ms |
| Mixed adds/removes/lists | 48 | 53.57 ms |
| Same-installation add/remove race | 32 | 44.69 ms |

These local workloads are fast. No performance threshold was imposed, and
network speed, repository size, cold caches, and other machines remain untested.
Aggregate timings in the JSON include fault injection; the synchronized Git
wrapper increases latency in the redundant-cleanup probe.

## Findings

1. **Medium: remove reports success when checkout deletion fails.** Add a
   repository, create a nonempty directory inside its checkout, chmod that
   directory to `0500`, then remove the repository as a non-root user. The CLI
   exits zero with `status: "removed"`, while protected files remain. The
   catalogue entry is already gone, list is empty, and retrying removal returns
   not-found. `internal/cli/remove/logic.go` discards the deletion error. A
   contextual failure/warning or a cleanup mechanism would make the remaining
   files visible and recoverable.
2. **Medium, accepted design limitation: duplicate adds can silently leak a
   redundant checkout.** Synchronize two real adds of the same reference and
   make a nonempty subdirectory in each checkout read-only before publication.
   Both adds return the same successful result and the catalogue contains one
   entry, but cleanup of the losing checkout fails and leaves an unregistered
   directory. `internal/cli/add/logic.go` discards that error. This is expressly
   accepted by [ADR 0005](adr/0005-catalogue-concurrency.md); its proposed future
   automatic cleanup is not implemented. The leftover needs manual deletion.

Both probes restore permissions and delete leftovers before completion. No
application behavior was changed as part of this QA task.

## Project checks

All seven packages were checked, including packages without test files:

| Check | Result | Wall time |
| --- | --- | ---: |
| `go test -count=1 ./...` | Passed | 445 ms |
| `go test -race -count=1 ./...` | Passed | 1,546 ms |
| `go vet ./...` | Passed | 41 ms |

The report retains their command output. Python compilation also passed.
