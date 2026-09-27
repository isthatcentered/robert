# Catalogue owns the reference type

Status: Accepted

## Context

The catalogue and CLI defined separate reference types with the same fields. Commands converted between them when reading or changing installations. The JSON adapter also had its own reference struct.

## Decision

- `catalog.Reference` is the single reference type used by catalogue entries, CLI commands, and Git operations.
- The catalogue package owns reference validation and equality rules as part of the domain interface.
- The JSON adapter serializes `catalog.Reference` directly, preserving the `type` and `value` field names.

## Consequences

- Commands use catalogue references directly without type conversions.
