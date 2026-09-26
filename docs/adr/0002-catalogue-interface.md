# 0002: Catalogue interface

## Status

Proposed. The catalogue interface direction is accepted; detailed contracts are being resolved through a design interview.

## Context

The current configuration store hides file access, but CLI commands still search entries, detect ambiguous selections, manipulate document slices, and save whole documents. Repository values and JSON preservation are also coupled in a CLI helper package.

## Accepted direction

- Introduce a separate catalogue module that owns its interface and domain types.
- Use Installation for a saved catalogue entry, consistent with the domain glossary.
- Represent a selection with a Repository and an optional Reference. An omitted reference selects all installations of that repository.
- Provide InstallDir, Find, Register, and Unregister operations. Mutations persist before returning success.
- Find returns all matches in saved order. Unregister requires exactly one match, validates its saved checkout path, persists its removal, and returns the removed installation.
- Parse accepted repository input forms into a domain Repository value through the catalogue module.
- Provide a filesystem adapter for the live CLI and a memory adapter for tests, with shared domain rules.
- Keep Git operations and checkout creation/deletion in the command workflows.
- Preserve the lifecycle described in ADR 0001, including missing-configuration behavior and unknown saved fields. The glossary now calls the saved installation collection the catalogue; the existing on-disk schema and CLI wording are not renamed by this decision.

## Open decisions

- Repository identity and normalization rules.
- Registration when an installation with the same repository and reference already exists.
- Concurrent operations and the consistency guarantees of mutations.
- Domain value validation and treatment of malformed saved entries.
- Representation of unknown saved metadata and generation of command output.
- Error types and translation into existing CLI errors.
- Adapter composition, shared rules, and contract testing.

## Consequences

Commands will depend on catalogue operations instead of configuration documents. Storage encoding and collection manipulation move behind the module boundary. Implementation will follow once the detailed contracts are agreed.
