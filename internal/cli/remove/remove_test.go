package remove

import (
	"strings"
	"testing"
)

func TestParseRepositoryAndReferenceForms(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		url   string
		kind  string
		value string
	}{
		{"shorthand", []string{"owner/repo"}, "https://github.com/owner/repo.git", "", ""},
		{"branch", []string{"owner/repo", "--branch=feature/x"}, "https://github.com/owner/repo.git", "branch", "feature/x"},
		{"single dash", []string{"owner/repo", "-branch", "main"}, "https://github.com/owner/repo.git", "branch", "main"},
		{"tag", []string{"https://example.com/repo.git", "--tag", "V1"}, "https://example.com/repo.git", "tag", "V1"},
		{"scp", []string{"git@example.com:team/repo.git", "--branch", "main"}, "git@example.com:team/repo.git", "branch", "main"},
		{"commit", []string{"ssh://git@example.com/team/repo.git", "--commit", strings.Repeat("A", 40)}, "ssh://git@example.com/team/repo.git", "commit", strings.Repeat("a", 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArgs(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if got.URL != tc.url {
				t.Fatalf("URL = %q, want %q", got.URL, tc.url)
			}
			if tc.kind == "" {
				if got.Reference != nil {
					t.Fatalf("unexpected reference: %+v", got.Reference)
				}
				return
			}
			if got.Reference == nil || got.Reference.Type != tc.kind || got.Reference.Value != tc.value {
				t.Fatalf("reference = %+v, want %s:%s", got.Reference, tc.kind, tc.value)
			}
		})
	}
}

func TestParseRejectsInvalidSelection(t *testing.T) {
	cases := [][]string{
		nil,
		{"repo"},
		{"./local/repo"},
		{"owner/repo", "--branch", "main", "--tag", "v1"},
		{"owner/repo", "--branch", "main", "--branch", "main"},
		{"owner/repo", "--commit", "abc"},
		{"owner/repo", "--tag="},
		{"owner/repo", "--branch"},
		{"owner/repo", "--branch", "--tag", "v1"},
		{"owner/repo", "--unknown", "main"},
		{"owner/repo", "extra", "--branch", "main"},
		{"owner/repo", "--branch", "main", "extra"},
		{"owner/repo", "--", "--branch", "main"},
	}
	for _, args := range cases {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("accepted invalid arguments %v", args)
		}
	}
}

func TestParseHelp(t *testing.T) {
	for _, args := range [][]string{
		{"-h"},
		{"--help"},
		{"owner/repo", "--help"},
		{"owner/repo", "--branch", "main", "-h"},
	} {
		got, err := parseArgs(args)
		if err != nil || !got.Help {
			t.Errorf("parseArgs(%v) = %+v, %v; want help", args, got, err)
		}
	}
}
