package update

import (
	"context"
	"fmt"
	"slices"

	"github.com/isthatcentered/robert/internal/cli/problem"
)

const Usage = "robert update"

const Help = `Update the repositories in your library to the latest version.

Usage:
  robert update

Options:
  -h, --help   Show help

Tag and commit installations are ignored. Updates use fast-forward-only pulls.
Branches that are already up to date are omitted from the results.

Example:
  robert update`

func Handle(ctx context.Context, args []string, logic Logic) (any, error) {
	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		return Help, nil
	}
	if len(args) != 0 {
		return nil, problem.New(fmt.Sprintf("unexpected argument %q", args[0]), map[string]any{"usage": Usage})
	}
	return logic.Update(ctx)
}
