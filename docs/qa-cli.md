# CLI QA — 2026-09-26

**Normal add/remove operations worked and were fast. Two cleanup issues found.**

| Result | Outcome |
| --- | --- |
| CLI calls | 1,057 in 3.96 seconds, including setup and cleanup |
| Scenarios | 25 passed; 2 reproduced cleanup issues |
| Concurrency | No lost updates, duplicate entries, or deadlocks observed |
| Project checks | Tests, race detector, `go vet`, and Python compilation passed |
| Cleanup | All test configurations and repositories removed; real user data untouched |

## Speed

| Workload | Median time |
| --- | ---: |
| Add | 18 ms |
| Remove | 1.7 ms |
| List | 1.3 ms |
| 16 concurrent adds | 42 ms for the whole batch |
| 16 concurrent removes | 7 ms for the whole batch |
| 48 mixed concurrent commands | 54 ms for the whole batch |

Measured on Linux with tiny local Git repositories and warm caches. Internet
clones, large repositories, cold caches, and macOS runtime behavior were not tested.

## Issues

| Severity | Trigger | What happens |
| --- | --- | --- |
| Medium | Permissions prevent checkout deletion | Remove reports success, but files remain and the catalogue entry is gone. |
| Medium | Cleanup fails after concurrent duplicate adds | Both adds report success, but the redundant checkout remains unregistered. This is an [accepted design limitation](adr/0005-catalogue-concurrency.md). |

Both cases silently leave files requiring manual cleanup. Neither was fixed
in this QA task. Reproduction steps are in the [detailed results](qa/cli-2026-09-26.json).

## Rerun

Requires Go, Git, and Python 3.9+. Run as a non-root user to include permission probes.

```sh
python3 scripts/qa_cli.py --workers 16 --rounds 5 --report /tmp/robert-qa-results.json
```

Uses isolated temporary homes and real local Git repositories. Covers branches,
tags, commits, duplicates, concurrent add/remove/list, invalid inputs, and storage
failures; verifies JSON, catalogue contents, Git checkout state, and leftovers.

Exit codes: **0** = clean, **1** = test/setup/cleanup failure, **2** = issues found.
The JSON report contains each invocation and timing; rerunning replaces it.

Tested revision: `71aa25d`. [Recorded evidence and full timing breakdown](qa/cli-2026-09-26.json).
