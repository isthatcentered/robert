# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues in `isthatcentered/robert`. Use the `gh` CLI for all operations. Run commands from this clone so `gh` selects the repository from its Git remote.

## Conventions

- **Create an issue:** `gh issue create --title "..." --body-file <file>`. Use a file for multiline bodies.
- **Read an issue:** `gh issue view <number> --comments`. Use `--json number,title,body,labels,comments` when structured output is needed.
- **List issues:** `gh issue list --state open --json number,title,body,labels,comments`, with `--label` or other filters as needed.
- **Comment:** `gh issue comment <number> --body-file <file>`.
- **Apply or remove labels:** `gh issue edit <number> --add-label "..."` or `--remove-label "..."`.
- **Close:** `gh issue close <number> --comment "..."`.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo later treats external PRs as feature requests.)_

When set to `yes`, use the corresponding `gh pr` commands. GitHub shares one number space across issues and PRs; resolve an ambiguous `#<number>` by checking the PR, then the issue.

## Skill instructions

- When a skill says "publish to the issue tracker", create a GitHub issue.
- When a skill says "fetch the relevant ticket", read the GitHub issue and its comments.

## Wayfinding operations

A map is one issue labelled `wayfinder:map`. Child tickets are linked as GitHub sub-issues when available; otherwise, list them in the map body and put `Part of #<map>` in each child body. Child labels use `wayfinder:<type>`.

Represent blockers with GitHub issue dependencies when available. For the dependency API, `issue_id` is the blocker's numeric database ID, not its issue number. Otherwise, put `Blocked by: #<number>` at the top of the child body. An unblocked ticket has no open blockers and no assignee. Claim it with `gh issue edit <number> --add-assignee @me`. Record the result in a comment, close the child, and update the map's decisions.
