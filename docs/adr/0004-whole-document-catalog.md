# 0004: Whole-document catalog storage

## Status

Accepted. Supersedes the operation-level interface proposed in ADR 0002 and the saved-field validation requirement in ADR 0003.

## Context

Commands depend on the JSON configuration store directly and repeat storage error handling. We want one storage boundary while leaving catalogue manipulation in the operation slices.

## Decision

- Introduce `internal/catalog.Catalog` with `Read() (Document, error)` and `Write(Document) error`. Both methods operate on the whole configuration.
- Expose `Document`, `Entry`, and `Reference` as plain values without JSON tags. The document retains version, installation directory, and saved installations.
- Implement `JSONFileCatalog`, which owns the file location, defaults, private JSON structs, decoding, encoding, and temporary-file replacement. Keep the existing on-disk schema and private file permissions.
- A missing file returns the default document without writing it. There is no separate existence result. Removing from a missing catalogue therefore reports an installation not found, just like an empty catalogue.
- Trust saved fields without additional validation on read or write. JSON decoding failures and filesystem errors still return storage operation, file path, and underlying cause. CLI error translation adds correction hints centrally.
- Retain the removal workflow's nonempty, absolute checkout-path precondition from ADR 0001 before changing the catalogue or deleting a directory.
- Commands retain matching, duplicate detection, document mutation, list filtering and ordering, Git operations, checkout management, and decisions about when to write.
- CLI response types own their JSON output tags independently of the private storage schema. Unknown saved fields remain ignored on read and dropped on subsequent writes.
- Retain the existing no-locking policy; concurrent read/modify/write operations can overwrite each other's updates.

## Consequences

Slices depend on the catalog interface and no longer know the configuration file path or storage format. Tests can provide a whole-document fake. This boundary centralizes storage rather than catalogue manipulation; higher-level operations can be introduced separately if needed.
