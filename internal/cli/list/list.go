package list

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

const Usage = "robert list [--search <text>] [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]"

type Selection struct {
	Search    string
	Reference *repository.Reference
	Help      bool
}

func Handle(args []string, logic Logic) (any, error) {
	selection, err := parseArgs(args)
	if err != nil {
		return nil, problem.New(err.Error(), map[string]any{"usage": Usage})
	}
	if selection.Help {
		return map[string]string{"usage": Usage}, nil
	}
	return logic.List(selection)
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
			ref := repository.Reference{Type: kind, Value: value}
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
	return result, nil
}
