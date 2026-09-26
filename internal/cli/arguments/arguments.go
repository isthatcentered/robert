package arguments

import (
	"fmt"
	"regexp"
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
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return Selection{Help: true}, nil
		}
	}
	if len(args) == 0 {
		return Selection{}, fmt.Errorf("repository is required")
	}
	url, err := repository.NormalizeURL(args[0])
	if err != nil {
		return Selection{}, err
	}
	result := Selection{URL: url}
	for i := 1; i < len(args); i++ {
		flag, value, hasEquals := strings.Cut(args[i], "=")
		kind := strings.TrimPrefix(flag, "--")
		if flag != "--branch" && flag != "--tag" && flag != "--commit" {
			return Selection{}, fmt.Errorf("unexpected argument %q: use --branch, --tag, or --commit", args[i])
		}
		if result.Reference != nil {
			return Selection{}, fmt.Errorf("select only one reference; flags cannot be repeated or combined")
		}
		if !hasEquals {
			i++
			if i >= len(args) {
				return Selection{}, fmt.Errorf("%s requires a value", flag)
			}
			value = args[i]
		}
		if value == "" || strings.HasPrefix(value, "--") {
			return Selection{}, fmt.Errorf("%s requires a nonempty value", flag)
		}
		if kind == "commit" {
			if !commitID.MatchString(value) {
				return Selection{}, fmt.Errorf("--commit requires a full 40-character hexadecimal ID")
			}
			value = strings.ToLower(value)
		}
		result.Reference = &repository.Reference{Type: kind, Value: value}
	}
	return result, nil
}
