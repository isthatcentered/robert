package catalog

import "testing"

func TestRepositoryURL(t *testing.T) {
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
		u, err := ParseRepositoryURL(remote)
		if err != nil {
			t.Fatal(err)
		}
		if string(u) != remote {
			t.Errorf("ParseRepositoryURL(%q) = %q", remote, u)
		}
		if got := u.SearchPath(); got != "group/team/repo" {
			t.Errorf("%q.SearchPath() = %q", u, got)
		}
	}
	short, err := ParseRepositoryURL("team/repo")
	if err != nil || short != "https://github.com/team/repo.git" || short.SearchPath() != "team/repo" {
		t.Errorf("ParseRepositoryURL(team/repo) = %q, %v", short, err)
	}
}

func TestParseRepositoryURLRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{
		"", "repo", "./local/repo", "../repo", "file:///tmp/repo",
		"https://example.com/", "git@example.com:", "https://example.com/\x00repo",
	} {
		if _, err := ParseRepositoryURL(input); err == nil {
			t.Errorf("ParseRepositoryURL(%q) accepted invalid input", input)
		}
	}
}
