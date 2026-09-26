# 0001: Add and remove lifecycle

## Status

Accepted

## Context

Robert saves installations in a configuration file and places each checkout in its own directory. The initial CLI specification named the removal command `delete`, while the intended command name is `remove`.

## Decision

- The removal command is `remove`.
- A successful `remove` returns JSON with `status: "removed"` and the saved installation fields, including `addedAt` and any other fields in that entry.
- The CLI performs these operations directly for now. A future daemon may take ownership of them.
- On the first successful `add`, Robert creates the configuration file after installing the checkout. If installation fails, Robert does not create the configuration file.
- An unflagged `add` resolves the remote's current default branch. If that branch differs from the reference in an existing installation, it creates a separate installation.
- The configuration is authoritative for duplicate detection. Repeating an `add` for the same URL and resolved reference returns `already_added` and the saved path even when the checkout directory is missing.
- If saving the configuration after a successful checkout fails, `add` attempts to remove that checkout and reports the save error. If cleanup also fails, the error includes the remaining path.
- `remove` saves the configuration without the selected installation before deleting its checkout directory.
- `remove` requires the saved checkout path to be nonempty and absolute before changing the configuration. An invalid path returns an error and leaves the entry intact. It otherwise trusts the path in the configuration without checking its parent or directory name.
- Once the configuration change is saved, `remove` reports ordinary success even if directory deletion fails. The response returns `status: "removed"` and the saved installation details, without mentioning a remaining directory. `removed` is only a result value; it is not a status stored in the configuration. A remaining directory is no longer an installation.

## Consequences

- A failed first installation does not create an empty configuration file.
- A failed directory deletion can leave an untracked checkout directory for a future cleanup operation.
- The CLI needs operation-specific slices now; daemon slices can be added when the daemon is introduced.
