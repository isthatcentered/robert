package list

import (
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	selection, err := parseArgs([]string{"--commit", strings.Repeat("A", 40), "--search", "  Acme/API  "})
	if err != nil {
		t.Fatal(err)
	}
	if selection.Search != "Acme/API" || selection.Reference.Type != "commit" || selection.Reference.Value != strings.Repeat("a", 40) {
		t.Fatalf("unexpected selection: %+v", selection)
	}
	for _, flags := range [][]string{nil, {"--search="}, {"--search", " \t"}} {
		selection, err := parseArgs(flags)
		if err != nil || selection.Search != "" {
			t.Fatalf("empty search = %+v, %v", selection, err)
		}
	}
}

func TestParseRejectsInvalidFlags(t *testing.T) {
	for _, flags := range [][]string{
		{"acme"}, {"--search"}, {"--search", "--branch", "main"},
		{"--search=a", "--search=b"}, {"--search=a", "extra"}, {"--unknown"},
		{"--branch"}, {"--branch="}, {"--tag", "--branch", "main"},
		{"--branch=main", "--tag=v1"}, {"--branch=main", "--branch=main"},
		{"--tag", "bad tag"}, {"--commit=abc"}, {"--commit", strings.Repeat("g", 40)},
		{"--", "acme"},
	} {
		if _, err := parseArgs(flags); err == nil {
			t.Errorf("accepted invalid flags %v", flags)
		}
	}
}

func TestHelpDoesNotReadCatalogue(t *testing.T) {
	for _, flags := range [][]string{{"-h"}, {"--help"}} {
		result, err := Handle(flags, Logic{})
		if err != nil {
			t.Fatal(err)
		}
		if result.(string) != Help {
			t.Fatalf("help = %v", result)
		}
	}
}
