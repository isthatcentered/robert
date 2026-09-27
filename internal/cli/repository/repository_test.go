package repository

import (
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
