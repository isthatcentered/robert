# 0003: Listing installations and validating configuration

## Status

Accepted. The saved-field validation requirement is superseded by [ADR 0004](0004-whole-document-catalog.md); saved configuration is now trusted.

## Context

Users need to discover saved installations by repository namespace/name and selected reference. The existing configuration loader validates only part of the document and explicitly preserves unknown JSON fields. Listing must not silently hide malformed entries or report them as an empty catalogue.

## Decision

- Add `robert list [--search <text>] [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]`.
- Search trims surrounding whitespace and performs a case-insensitive, literal substring match against the remote's full repository path, including nested namespaces. Exclude the host, URL query/fragment, and trailing `.git`; keep the saved URL unchanged in output. An empty search matches everything.
- Reference flags are mutually exclusive and cannot repeat. Branch and tag values match exactly, including case. Commit IDs require 40 hexadecimal characters and compare independently of hex casing. Search and reference filters must both match.
- Return a JSON array of `{url, reference: {type, value}, path}` objects, one per matching installation.
- Sort by saved URL ascending, then addition timestamp oldest first, then reference type, reference value, and checkout path ascending. Compare timestamps chronologically, including offsets and fractional seconds. String ordering is case-sensitive.
- A missing configuration file, empty catalogue, or no matches returns `[]`. Listing never writes configuration, checks checkout existence, or contacts remote repositories.
- Shared configuration handling validates the whole document on read and before write, before any list filtering. Require version 1, an absolute installation directory, an array of installations, and each entry's full remote URL, absolute checkout path, valid branch/tag/commit reference, and RFC3339 addition timestamp. Checkout paths may point to missing directories.
- Invalid or unreadable configuration returns a JSON error with the file path, cause, and a correction hint. Invalid saved fields identify the entry and field. Failed validation before writing leaves the existing file unchanged.
- Duplicate prevention belongs to `add`, which retains the successful `already_added` response and does not create a second entry. `list` and shared field validation do not enforce installation uniqueness.
- Remove explicit preservation of unknown JSON fields at document, installation, and reference levels. Ignore unknown fields on read and omit them on subsequent writes. A read alone does not rewrite the file. This supersedes the preservation requirement in ADRs 0001 and 0002.
- No locking or retry protocol is introduced. Concurrent file access remains an accepted limitation; existing temporary-file replacement on save is retained.

## Consequences

All commands now reject malformed saved fields, even on unrelated entries. Users must correct invalid configuration before retrying. Unknown metadata disappears when a command next writes the catalogue. The future catalogue module in ADR 0002 must retain these behaviors, while `list` owns its output projection and ordering.
