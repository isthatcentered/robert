package arguments

import (
	"flag"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Selection struct {
	URL       string
	Reference *repository.Reference
	Help      bool
}

var commitID = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func Parse(args []string) (Selection, error) {
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
	flags := flag.NewFlagSet("repository", flag.ContinueOnError)
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
