package list

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

const Usage = "robert list [--search <text>] [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]"

const Help = `List saved repository checkouts.

Usage:
  robert list [--search <text>] [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]

Options:
  --search     Match text in the repository owner/name, ignoring case
  --branch     Filter by an exact branch
  --tag        Filter by an exact tag
  --commit     Filter by an exact full 40-character commit ID
  -h, --help   Show help

Search and one reference filter can be combined.

Examples:
  robert list
  robert list --search acme/api
  robert list --branch main
  robert list --search acme --tag v1.0.0`

type Selection struct {
	Search    string
	Reference *catalog.Reference
	Help      bool
	Filtered  bool
}

type Output struct {
	Results  []Result
	Filtered bool
}

func Handle(args []string, logic Logic) (any, error) {
	selection, err := parseArgs(args)
	if err != nil {
		return nil, problem.New(err.Error(), map[string]any{"usage": Usage})
	}
	if selection.Help {
		return Help, nil
	}
	results, err := logic.List(selection)
	if err != nil {
		return nil, err
	}
	return Output{Results: results, Filtered: selection.Filtered}, nil
}

func parseArgs(args []string) (Selection, error) {
	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		return Selection{Help: true}, nil
	}
	var result Selection
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	searchSet := false
	flags.Func("search", "search the repository namespace and name", func(value string) error {
		if searchSet {
			return fmt.Errorf("--search cannot be repeated")
		}
		if strings.HasPrefix(value, "--") {
			return fmt.Errorf("--search requires a value; use --search= for an empty search")
		}
		searchSet = true
		result.Search = strings.TrimSpace(value)
		return nil
	})
	for _, kind := range []string{"branch", "tag", "commit"} {
		flags.Func(kind, "filter by exact repository "+kind, func(value string) error {
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
	if err := flags.Parse(args); err != nil {
		return Selection{}, err
	}
	if flags.NArg() != 0 {
		return Selection{}, fmt.Errorf("unexpected argument %q: use --search, --branch, --tag, or --commit", flags.Arg(0))
	}
	result.Filtered = searchSet || result.Reference != nil
	return result, nil
}
