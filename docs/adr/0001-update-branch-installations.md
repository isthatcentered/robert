# Update branch installations

Status: Accepted

## Context

Robert saves installations with a selected branch, tag, or commit reference. The `update` command updates saved branch installations in place. A checkout may have changed independently since it was added.

## Decisions

- Select branch installations by their saved reference in the catalogue. Ignore tag and commit installations.
- For each branch installation, check that the checkout is on the saved branch. A missing checkout or a different current branch is a failure for that installation.
- On the saved branch, run a fast-forward-only pull. If the pull fails, report that installation as failed. Do not add separate checks for dirty files, local-only commits, or changed remote/upstream settings; Git determines whether the pull succeeds.
- Attempt every branch installation even when one fails. Report branches whose HEAD advanced and all failed attempts, and return a nonzero exit status if any update fails.
- Omit successful pulls that leave HEAD unchanged. This includes an already-current branch and a branch that is only ahead of its remote.
- Render results in the same table style as `list`, with `REPOSITORY`, `REFERENCE`, and `CHECKOUT` columns plus a `RESULT` column. Follow the table with contextual details for each failure.
- If branch installations exist but none advance or fail, print `All branch checkouts are already up to date.`
- If the catalogue has no installations, print `No repositories installed. Run "robert add <repository>" to install one.` If it has installations but no branch installations, print `No branch-based repositories installed; nothing to update.`

## Consequences

- The catalogue selects which installations to visit, but the checkout's Git configuration controls the pull target. Robert does not verify that `origin` or upstream still matches the catalogue entry.
- A successful fast-forward pull may proceed with unrelated working-tree edits if Git permits it.
