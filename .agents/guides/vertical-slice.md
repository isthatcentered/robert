# Vertical slices

## Goal

Organize the CLI and daemon by operation so the code for `add`, `remove`, `list`, or `run` is easy to find. Keep application logic separate from command parsing, HTTP handling, and external systems. Define the interfaces an operation needs beside its logic; implement them in adapters for Git, databases, HTTP APIs, and other external dependencies. This makes the logic straightforward to test with a fake and lets an adapter change without rewriting the operation.

## Layout

```text
cmd/
  robert/main.go                 # CLI executable, construct dependencies and dispatch commands
  robertd/main.go                # daemon executable
internal/
  cli/
    add/
      add.go                    # command handler
      logic.go                  # CLI workflow
      dependencies.go           # interfaces used by that workflow
      adapters.go               # daemon HTTP client implementation
    remove/                     # same four-file pattern
    list/
    run/
  daemon/
    daemon.go                   # construct dependencies and register routes
    add/
      add.go                    # HTTP handler
      logic.go                  # application logic
      dependencies.go           # interfaces used by that logic
      adapters.go               # external-system implementations
    remove/                     # same four-file pattern
    list/
    run/
```

Each slice is a separate Go package. Its named file (`add.go`, `run.go`, etc.) is the entry point for that operation. Go imports the **package**, not that file: `cmd/robert/main.go` constructs CLI dependencies and imports CLI slices directly. `internal/daemon/daemon.go` imports daemon slices, and `cmd/robertd/main.go` calls `daemon.Run`.

## Example: daemon `add`

```go
// internal/daemon/add/dependencies.go
package add

import "context"

type Repository interface {
    Add(ctx context.Context, name string) error
}
```

```go
// internal/daemon/add/logic.go
package add

import (
    "context"
    "errors"
    "strings"
)

type Logic struct{ repo Repository }

func NewLogic(repo Repository) Logic { return Logic{repo: repo} }

func (l Logic) Add(ctx context.Context, name string) error {
    name = strings.TrimSpace(name)
    if name == "" {
        return errors.New("name is required")
    }
    return l.repo.Add(ctx, name)
}
```

```go
// internal/daemon/add/adapters.go
package add

import (
    "context"
    "database/sql"
)

type SQLRepository struct{ DB *sql.DB }

func (r SQLRepository) Add(ctx context.Context, name string) error {
    _, err := r.DB.ExecContext(ctx, "INSERT INTO items (name) VALUES (?)", name)
    return err
}
```

`add.go` decodes `POST /add`, calls `Logic.Add`, and writes the HTTP response. `daemon.go` creates the shared database connection, constructs `SQLRepository`, and registers the handler. The logic knows only the `Repository` interface; a test can pass a small fake instead of using a database.

The CLI has the same boundary with a different dependency: `internal/cli/add/dependencies.go` declares the daemon operation the CLI workflow needs, while `adapters.go` implements it with HTTP. Each CLI command owns its argument parsing in its named entry-point file and its parsing tests alongside it; do not share argument parsers across slices. The CLI slice handles arguments and output; the daemon slice owns the operation's business rules. Share long-lived resources such as an HTTP client or database pool during composition, rather than opening one per request.
