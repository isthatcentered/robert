package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/isthatcentered/robert/internal/catalog"
)

func TestListSearchAndReferenceCombinations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sha := "1234567890abcdef1234567890abcdef12345678"
	doc := catalog.Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []catalog.Entry{}}
	entry := func(remote, kind, value, path string) catalog.Entry {
		return catalog.Entry{URL: remote, Reference: catalog.Reference{Type: kind, Value: value}, Path: filepath.Join(home, path), AddedAt: "2026-09-26T12:00:00Z"}
	}
	api := "https://github.com/acme/api.git"
	web := "https://github.com/acme/web.git"
	tools := "https://github.com/tools/api.git"
	doc.Repositories = []catalog.Entry{
		entry(tools, "branch", "main", "tools-api-main"),
		entry(web, "commit", sha, "acme-web-commit"),
		entry(api, "tag", "v1.0", "acme-api-v1"),
		entry(api, "branch", "main", "acme-api-main"),
	}
	path := filepath.Join(home, ".robert")
	writeCatalog(t, path, doc)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		flags []string
		paths []string
	}{
		{"all", nil, []string{"acme-api-main", "acme-api-v1", "acme-web-commit", "tools-api-main"}},
		{"owner", []string{"--search", "ACME"}, []string{"acme-api-main", "acme-api-v1", "acme-web-commit"}},
		{"name", []string{"--search", "api"}, []string{"acme-api-main", "acme-api-v1", "tools-api-main"}},
		{"boundary and trimming", []string{"--search", "  ME/AP\t"}, []string{"acme-api-main", "acme-api-v1"}},
		{"empty", []string{"--search="}, []string{"acme-api-main", "acme-api-v1", "acme-web-commit", "tools-api-main"}},
		{"whitespace", []string{"--search", " \t\n"}, []string{"acme-api-main", "acme-api-v1", "acme-web-commit", "tools-api-main"}},
		{"branch", []string{"--branch", "main"}, []string{"acme-api-main", "tools-api-main"}},
		{"tag", []string{"--tag", "v1.0"}, []string{"acme-api-v1"}},
		{"commit", []string{"--commit", strings.ToUpper(sha)}, []string{"acme-web-commit"}},
		{"search and branch", []string{"--search", "acme", "--branch", "main"}, []string{"acme-api-main"}},
		{"search and tag", []string{"--tag", "v1.0", "--search=api"}, []string{"acme-api-v1"}},
		{"search and commit", []string{"--search", "web", "--commit", sha}, []string{"acme-web-commit"}},
		{"search must match commit too", []string{"--search", "api", "--commit", sha}, []string{}},
		{"both must match", []string{"--search", "tools", "--tag", "v1.0"}, []string{}},
		{"branch prefix", []string{"--branch", "mai"}, []string{}},
		{"branch case", []string{"--branch", "MAIN"}, []string{}},
		{"tag case", []string{"--tag", "V1.0"}, []string{}},
		{"tag prefix", []string{"--tag", "v1"}, []string{}},
		{"reference type", []string{"--tag", "main"}, []string{}},
		{"no match", []string{"--search", "missing"}, []string{}},
		{"host excluded", []string{"--search", "github"}, []string{}},
		{"suffix excluded", []string{"--search", ".git"}, []string{}},
		{"literal wildcard", []string{"--search", "a*"}, []string{}},
		{"literal regex", []string{"--search", "a.*"}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := listCommand(t, tc.flags...)
			paths := make([]string, 0, len(results))
			for _, result := range results {
				paths = append(paths, filepath.Base(stringField(t, result, "path")))
				if len(result) != 3 || len(result["reference"].(map[string]any)) != 2 {
					t.Fatalf("unexpected result fields: %v", result)
				}
				found := false
				for _, saved := range doc.Repositories {
					if saved.Path == result["path"] {
						assertField(t, result, "url", saved.URL)
						assertReference(t, result, saved.Reference.Type, saved.Reference.Value)
						found = true
					}
				}
				if !found {
					t.Fatalf("unknown installation: %v", result)
				}
			}
			if !reflect.DeepEqual(paths, tc.paths) {
				t.Fatalf("paths = %v, want %v", paths, tc.paths)
			}
		})
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("list changed the catalogue: %v", err)
	}
}

func TestListEmptyCatalogue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if results := listCommand(t); len(results) != 0 {
		t.Fatalf("missing catalogue returned %v", results)
	}
	path := filepath.Join(home, ".robert")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("list created a catalogue: %v", err)
	}
	writeCatalog(t, path, catalog.Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []catalog.Entry{}})
	if results := listCommand(t, "--branch", "main"); len(results) != 0 {
		t.Fatalf("empty catalogue returned %v", results)
	}
}

func TestListSortsChronologicallyAndBreaksTiesWithoutRejectingDuplicates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	doc := catalog.Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []catalog.Entry{}}
	// Chronological order differs from string order for both offsets and fractions.
	for i, data := range []struct{ kind, value, date string }{
		{"tag", "z", "2026-09-26T12:00:00+02:00"},
		{"branch", "main", "2026-09-26T10:00:00Z"},
		{"branch", "main", "2026-09-26T10:00:00Z"},
		{"branch", "next", "2026-09-26T10:00:00Z"},
		{"branch", "early", "2026-09-26T11:00:00+02:00"},
		{"branch", "fraction", "2026-09-26T10:00:00.1Z"},
		{"branch", "later", "2026-09-26T10:30:00Z"},
	} {
		path := filepath.Join(home, string(rune('a'+i)))
		doc.Repositories = append(doc.Repositories, catalog.Entry{URL: "https://example.com/team/api.git", Reference: catalog.Reference{Type: data.kind, Value: data.value}, Path: path, AddedAt: data.date})
	}
	path := filepath.Join(home, ".robert")
	for _, reverse := range []bool{false, true} {
		if reverse {
			for i, j := 0, len(doc.Repositories)-1; i < j; i, j = i+1, j-1 {
				doc.Repositories[i], doc.Repositories[j] = doc.Repositories[j], doc.Repositories[i]
			}
		}
		writeCatalog(t, path, doc)
		results := listCommand(t)
		var paths []string
		for _, result := range results {
			paths = append(paths, filepath.Base(stringField(t, result, "path")))
		}
		if want := []string{"e", "b", "c", "d", "a", "f", "g"}; !reflect.DeepEqual(paths, want) {
			t.Fatalf("paths = %v, want %v", paths, want)
		}
	}
}

func TestAllCommandsReportMalformedJSONBeforeFilteringOrMutation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".robert")
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"list", "--search", "other"}, {"add", "other/repo", "--branch", "main"}, {"remove", "other/repo"}} {
		result := command(t, 1, args...)
		assertField(t, result, "error", "failed to read configuration")
		failure := result["context"].(map[string]any)
		assertField(t, failure, "path", path)
		cause := stringField(t, failure, "cause")
		if !strings.Contains(cause, "invalid JSON") || stringField(t, failure, "hint") == "" {
			t.Fatalf("missing actionable error context: %v", failure)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("command changed invalid catalogue: %v", err)
	}
}

func TestListReportsInvalidJSONAndReadErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".robert")
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []bool{false, true} {
		if directory {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
		}
		result := command(t, 1, "list")
		failure := result["context"].(map[string]any)
		assertField(t, failure, "path", path)
		if stringField(t, failure, "cause") == "" || stringField(t, failure, "hint") == "" {
			t.Fatalf("missing read error context: %v", result)
		}
	}
}

func listCommand(t *testing.T, flags ...string) []map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	args := append([]string{"list"}, flags...)
	if code := run(context.Background(), args, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("robert %v exited %d: %s", args, code, stderr.String())
	}
	var results []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil || results == nil {
		t.Fatalf("expected JSON array, got %s: %v", stdout.String(), err)
	}
	return results
}

func writeCatalog(t *testing.T, path string, doc catalog.Document) {
	t.Helper()
	if err := (catalog.JSONFileCatalog{Path: path}).Update(func(latest *catalog.Document) error { *latest = doc; return nil }); err != nil {
		t.Fatal(err)
	}
}
