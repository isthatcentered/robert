# Catalogue owns the repository URL type

Status: Accepted

## Context

The CLI parsed repository arguments as strings for `add` and `remove`. The `list` command separately extracted a search path from saved URL strings. Catalogue entries also held plain strings.

## Decision

- `catalog.RepositoryURL` is the domain type for a remote repository URL.
- `ParseRepositoryURL` validates remote URLs and expands `owner/repo` shorthand at command input.
- `RepositoryURL.SearchPath` derives the namespace and repository name for catalogue search.
- Catalogue JSON reads trust saved URL strings and do not validate them again.

## Consequences

- CLI selections, results, and catalogue entries share the URL type.
- Git adapters convert the URL to a string when invoking Git.
- The saved JSON format remains a string URL.
