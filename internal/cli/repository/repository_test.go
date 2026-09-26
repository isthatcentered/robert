package repository

import (
	"strings"
	"testing"
)

func TestSearchPath(t *testing.T) {
	for _, remote := range []string{
		"https://example.com/group/team/repo.git",
		"http://example.com/group/team/repo",
		"ssh://git@example.com:2222/group/team/repo.git",
		"git://example.com/group/team/repo.git/",
		"git@example.com:group/team/repo.git",
		"git@example.com:/group/team/repo.git",
		"https://example.com/group/team/repo.git?ignored=yes#ignored",
		"https://example.com/group/team/r%65po.git",
	} {
		if _, err := NormalizeURL(remote); err != nil {
			t.Fatal(err)
		}
		if got := SearchPath(remote); got != "group/team/repo" {
			t.Errorf("SearchPath(%q) = %q", remote, got)
		}
	}
}

func TestReferenceValidation(t *testing.T) {
	for _, kind := range []string{"branch", "tag"} {
		for _, value := range []string{"main", "feature/x", "v1.0", "UPPER", "@", "release/é"} {
			if err := (Reference{kind, value}).Validate(); err != nil {
				t.Errorf("rejected %s %q: %v", kind, value, err)
			}
		}
		for _, value := range []string{"", "bad name", "a\tb", "a\x00b", "a\x7fb", "a..b", "a.lock/b", ".hidden", "a/.hidden", "a@{b", "a//b", "/a", "a/", "a.", "a~b", "a^b", "a:b", "a?b", "a*b", "a[b", "a\\b"} {
			if err := (Reference{kind, value}).Validate(); err == nil {
				t.Errorf("accepted %s %q", kind, value)
			}
		}
	}
	for _, ref := range []Reference{{"branch", "HEAD"}, {"branch", "-main"}, {"commit", "abc"}, {"commit", strings.Repeat("g", 40)}, {"other", "main"}} {
		if err := ref.Validate(); err == nil {
			t.Errorf("accepted reference %+v", ref)
		}
	}
	if err := (Reference{"commit", strings.Repeat("A", 40)}).Validate(); err != nil {
		t.Fatal(err)
	}
}
