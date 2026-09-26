package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isthatcentered/robert/internal/cli/repository"
)

func validDocument(home string) Document {
	doc := New(home)
	doc.Repositories = []Entry{{URL: "https://example.com/team/api.git", Path: filepath.Join(home, "missing-checkout"), Reference: repository.Reference{Type: "branch", Value: "main"}, AddedAt: "2026-09-26T12:00:00.123Z"}}
	return doc
}

func TestInvalidFieldsAreRejectedOnReadAndBeforeWrite(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Document)
		field  string
	}{
		{"version", func(d *Document) { d.Version = 2 }, "version"},
		{"install directory", func(d *Document) { d.InstallDir = "relative" }, "installDir"},
		{"missing array", func(d *Document) { d.Repositories = nil }, "repositories"},
		{"empty URL", func(d *Document) { d.Repositories[0].URL = "" }, "url"},
		{"shorthand URL", func(d *Document) { d.Repositories[0].URL = "owner/repo" }, "url"},
		{"invalid URL", func(d *Document) { d.Repositories[0].URL = "file:///tmp/repo" }, "url"},
		{"missing remote path", func(d *Document) { d.Repositories[0].URL = "https://example.com/" }, "url"},
		{"empty checkout path", func(d *Document) { d.Repositories[0].Path = "" }, "path"},
		{"relative checkout path", func(d *Document) { d.Repositories[0].Path = "relative" }, "path"},
		{"NUL checkout path", func(d *Document) { d.Repositories[0].Path = "/tmp/\x00" }, "path"},
		{"missing reference", func(d *Document) { d.Repositories[0].Reference = repository.Reference{} }, "reference"},
		{"reference type", func(d *Document) { d.Repositories[0].Reference.Type = "revision" }, "reference"},
		{"empty branch", func(d *Document) { d.Repositories[0].Reference.Value = "" }, "reference"},
		{"invalid branch", func(d *Document) { d.Repositories[0].Reference.Value = "a..b" }, "reference"},
		{"short commit", func(d *Document) { d.Repositories[0].Reference = repository.Reference{Type: "commit", Value: "abc"} }, "reference"},
		{"missing timestamp", func(d *Document) { d.Repositories[0].AddedAt = "" }, "addedAt"},
		{"invalid timestamp", func(d *Document) { d.Repositories[0].AddedAt = "yesterday" }, "addedAt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			store := Store{Home: home, Path: filepath.Join(home, ".robert")}
			doc := validDocument(home)
			if err := store.Save(doc); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(store.Path)
			if err != nil {
				t.Fatal(err)
			}
			tc.change(&doc)
			if err := store.Save(doc); err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("Save error = %v, want field %q", err, tc.field)
			}
			after, err := os.ReadFile(store.Path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("invalid save modified file: %v", err)
			}
			data, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(store.Path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, exists, err := store.Load(); !exists || err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("Load exists=%v error=%v, want field %q", exists, err, tc.field)
			}
		})
	}
}

func TestLoadRejectsMalformedStructure(t *testing.T) {
	home := t.TempDir()
	store := Store{Home: home, Path: filepath.Join(home, ".robert")}
	for _, raw := range []string{
		`null`, `[]`, `{}`, `{"version":1,"installDir":"/tmp","repositories":null}`,
		`{"version":1,"installDir":"/tmp","repositories":[null]}`,
		`{"version":1,"installDir":"/tmp","repositories":[{"url":42}]}`,
		`{"version":1,"installDir":"/tmp","repositories":{}}`,
		`{"version":1,"installDir":"/tmp","repositories":[]} {}`,
	} {
		if err := os.WriteFile(store.Path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.Load(); err == nil {
			t.Errorf("accepted broken configuration %s", raw)
		}
	}
}

func TestUnknownFieldsAreIgnoredAndDroppedOnWrite(t *testing.T) {
	home := t.TempDir()
	store := Store{Home: home, Path: filepath.Join(home, ".robert")}
	raw := `{"version":1,"installDir":"/tmp/robert","custom":true,"repositories":[{"url":"git@example.com:group/api.git","path":"/tmp/missing","addedAt":"2026-09-26T12:00:00Z","note":"ignored","reference":{"type":"tag","value":"v1","source":"ignored"}}]}`
	if err := os.WriteFile(store.Path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	doc, _, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.Path)
	if err != nil || string(before) != raw {
		t.Fatalf("Load modified file: %v", err)
	}
	if err := store.Save(doc); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"custom", "note", "source"} {
		if bytes.Contains(after, []byte(field)) {
			t.Errorf("retained unknown field %s", field)
		}
	}
	if _, _, err := store.Load(); err != nil {
		t.Fatalf("saved invalid configuration: %v", err)
	}
}

func TestSaveDoesNotOwnDuplicateInvariant(t *testing.T) {
	home := t.TempDir()
	store := Store{Home: home, Path: filepath.Join(home, ".robert")}
	doc := validDocument(home)
	doc.Repositories = append(doc.Repositories, doc.Repositories[0])
	if err := store.Save(doc); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := store.Load()
	if err != nil || len(loaded.Repositories) != 2 {
		t.Fatalf("Load = %+v, %v", loaded, err)
	}
}
