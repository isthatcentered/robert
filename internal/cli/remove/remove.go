package remove

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

const Usage = "robert remove <repo> [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]"

const Help = `Remove a repository checkout from the catalogue.

Usage:
  robert remove <repo> [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]

Arguments:
  <repo>       owner/repo or a remote Git URL

Options:
  --branch     Select an exact branch
  --tag        Select an exact tag
  --commit     Select an exact commit by its full 40-character ID
  -h, --help   Show help

A reference is required when the repository has multiple saved checkouts.

Examples:
  robert remove acme/api
  robert remove https://github.com/isthatcentered/robert.git
  robert remove acme/api --branch main
  robert remove acme/api --tag v1.0.0
  robert remove acme/api --commit 0123456789abcdef0123456789abcdef01234567`

func Handle(args []string, logic Logic) (any, error) {
	selection, err := parseArgs(args)
	if err != nil {
		return nil, problem.New(err.Error(), map[string]any{"usage": Usage})
	}
	if selection.Help {
		return Help, nil
	}
	return logic.Remove(selection)
}

type Selection struct {
	URL       catalog.RepositoryURL
	Reference *catalog.Reference
	Help      bool
}

func parseArgs(args []string) (Selection, error) {
	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		return Selection{Help: true}, nil
	}
	if len(args) == 0 {
		return Selection{}, fmt.Errorf("repository is required")
	}
	url, err := catalog.ParseRepositoryURL(args[0])
	if err != nil {
		return Selection{}, err
	}
	result := Selection{URL: url}
	flags := flag.NewFlagSet("remove", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	for _, kind := range []string{"branch", "tag", "commit"} {
		flags.Func(kind, "select a repository "+kind, func(value string) error {
			if result.Reference != nil {
				return fmt.Errorf("select only one reference; flags cannot be repeated or combined")
			}
			if value == "" || strings.HasPrefix(value, "--") {
				return fmt.Errorf("--%s requires a nonempty value", kind)
			}
			ref := catalog.Reference{Type: kind, Value: value}
			if err := ref.Validate(); err != nil {
				return err
			}
			if kind == "commit" {
				ref.Value = strings.ToLower(value)
			}
			result.Reference = &ref
			return nil
		})
	}
	if err := flags.Parse(args[1:]); err != nil {
		return Selection{}, err
	}
	if flags.NArg() != 0 {
		return Selection{}, fmt.Errorf("unexpected argument %q: use --branch, --tag, or --commit", flags.Arg(0))
	}
	return result, nil
}
