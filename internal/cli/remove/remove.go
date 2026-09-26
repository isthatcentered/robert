package remove

import (
	"flag"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

const Usage = "robert remove <repo> [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]"

func Handle(args []string, logic Logic) (any, error) {
	selection, err := parseArgs(args)
	if err != nil {
		return nil, problem.New(err.Error(), map[string]any{"usage": Usage})
	}
	if selection.Help {
		return map[string]string{"usage": Usage}, nil
	}
	return logic.Remove(selection)
}

type Selection struct {
	URL       string
	Reference *repository.Reference
	Help      bool
}

var commitID = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func parseArgs(args []string) (Selection, error) {
	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		return Selection{Help: true}, nil
	}
	if len(args) == 0 {
		return Selection{}, fmt.Errorf("repository is required")
	}
	url, err := repository.NormalizeURL(args[0])
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
			if kind == "commit" {
				if !commitID.MatchString(value) {
					return fmt.Errorf("--commit requires a full 40-character hexadecimal ID")
				}
				value = strings.ToLower(value)
			}
			result.Reference = &repository.Reference{Type: kind, Value: value}
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
