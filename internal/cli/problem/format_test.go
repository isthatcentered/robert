package problem

import (
	"strings"
	"testing"

	"github.com/isthatcentered/robert/internal/cli/repository"
)

func TestFormatAmbiguousCheckoutError(t *testing.T) {
	err := New("multiple saved checkouts match this repository", map[string]any{
		"url":                "https://github.com/acme/api.git",
		"matchingReferences": []repository.Reference{{Type: "branch", Value: "main"}, {Type: "tag", Value: "v1.0.0"}},
		"hint":               "specify --branch, --tag, or --commit",
	})
	got := Format(err)
	for _, want := range []string{
		"error: multiple saved checkouts match this repository\n",
		"Repository:        https://github.com/acme/api.git\n",
		"Matches:           branch main, tag v1.0.0\n",
		"Hint:              Specify --branch, --tag, or --commit.\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted error missing %q:\n%s", want, got)
		}
	}
}
