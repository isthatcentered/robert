package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func validDocument(home string) Document {
	doc := Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []Entry{}}
	doc.Repositories = []Entry{{URL: "https://example.com/team/api.git", Path: filepath.Join(home, "missing-checkout"), Reference: Reference{Type: "branch", Value: "main"}, AddedAt: "2026-09-26T12:00:00.123Z"}}
	return doc
}

func TestSavedFieldsAreTrustedOnReadAndUpdate(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Document)
	}{
		{"version", func(d *Document) { d.Version = 2 }},
		{"install directory", func(d *Document) { d.InstallDir = "relative" }},
		{"missing array", func(d *Document) { d.Repositories = nil }},
		{"empty URL", func(d *Document) { d.Repositories[0].URL = "" }},
		{"shorthand URL", func(d *Document) { d.Repositories[0].URL = "owner/repo" }},
		{"invalid URL", func(d *Document) { d.Repositories[0].URL = "file:///tmp/repo" }},
		{"missing remote path", func(d *Document) { d.Repositories[0].URL = "https://example.com/" }},
		{"empty checkout path", func(d *Document) { d.Repositories[0].Path = "" }},
		{"relative checkout path", func(d *Document) { d.Repositories[0].Path = "relative" }},
		{"NUL checkout path", func(d *Document) { d.Repositories[0].Path = "/tmp/\x00" }},
		{"missing reference", func(d *Document) { d.Repositories[0].Reference = Reference{} }},
		{"reference type", func(d *Document) { d.Repositories[0].Reference.Type = "revision" }},
		{"empty branch", func(d *Document) { d.Repositories[0].Reference.Value = "" }},
		{"invalid branch", func(d *Document) { d.Repositories[0].Reference.Value = "a..b" }},
		{"short commit", func(d *Document) { d.Repositories[0].Reference = Reference{Type: "commit", Value: "abc"} }},
		{"missing timestamp", func(d *Document) { d.Repositories[0].AddedAt = "" }},
		{"invalid timestamp", func(d *Document) { d.Repositories[0].AddedAt = "yesterday" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			store := JSONFileCatalog{Home: home, Path: filepath.Join(home, ".robert")}
			doc := validDocument(home)
			if err := store.Update(func(latest *Document) error { *latest = doc; return nil }); err != nil {
				t.Fatal(err)
			}
			tc.change(&doc)
			if err := store.Update(func(latest *Document) error { *latest = doc; return nil }); err != nil {
				t.Fatalf("Update rejected authoritative configuration: %v", err)
			}
			loaded, err := store.Read()
			if err != nil || !reflect.DeepEqual(loaded, doc) {
				t.Fatalf("Read = %+v, %v; want %+v", loaded, err, doc)
			}
		})
	}
}

func TestReadReportsJSONDecodingErrors(t *testing.T) {
	home := t.TempDir()
	store := JSONFileCatalog{Home: home, Path: filepath.Join(home, ".robert")}
	for _, raw := range []string{
		`{`, `[]`,
		`{"version":1,"installDir":"/tmp","repositories":[{"url":42}]}`,
		`{"version":1,"installDir":"/tmp","repositories":{}}`,
		`{"version":1,"installDir":"/tmp","repositories":[]} {}`,
	} {
		if err := os.WriteFile(store.Path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Read(); err == nil {
			t.Errorf("accepted broken configuration %s", raw)
		}
	}
}

func TestUnknownFieldsAreIgnoredAndDroppedOnUpdate(t *testing.T) {
	home := t.TempDir()
	store := JSONFileCatalog{Home: home, Path: filepath.Join(home, ".robert")}
	raw := `{"version":1,"installDir":"/tmp/robert","custom":true,"repositories":[{"url":"git@example.com:group/api.git","path":"/tmp/missing","addedAt":"2026-09-26T12:00:00Z","note":"ignored","reference":{"type":"tag","value":"v1","source":"ignored"}}]}`
	if err := os.WriteFile(store.Path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.Path)
	if err != nil || string(before) != raw {
		t.Fatalf("Read modified file: %v", err)
	}
	if err := store.Update(func(latest *Document) error { *latest = doc; return nil }); err != nil {
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
	if _, err := store.Read(); err != nil {
		t.Fatalf("saved invalid configuration: %v", err)
	}
}

func TestUpdateDoesNotOwnDuplicateInvariant(t *testing.T) {
	home := t.TempDir()
	store := JSONFileCatalog{Home: home, Path: filepath.Join(home, ".robert")}
	doc := validDocument(home)
	doc.Repositories = append(doc.Repositories, doc.Repositories[0])
	if err := store.Update(func(latest *Document) error { *latest = doc; return nil }); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Read()
	if err != nil || len(loaded.Repositories) != 2 {
		t.Fatalf("Read = %+v, %v", loaded, err)
	}
}

func TestMissingFileReturnsDefaultsWithoutWriting(t *testing.T) {
	home := t.TempDir()
	store := NewJSONFileCatalog(home)
	doc, err := store.Read()
	if err != nil || doc.Version != 1 || doc.InstallDir != filepath.Join(home, ".agents", "robert") || doc.Repositories == nil || len(doc.Repositories) != 0 {
		t.Fatalf("Read = %+v, %v", doc, err)
	}
	if _, err := os.Stat(store.Path); !os.IsNotExist(err) {
		t.Fatalf("Read created file: %v", err)
	}
}

func TestReadAndUpdateErrorsRetainStorageContext(t *testing.T) {
	home := t.TempDir()
	store := JSONFileCatalog{Home: home, Path: home} // A directory cannot be read or replaced as a file.
	_, readErr := store.Read()
	updateErr := store.Update(func(latest *Document) error { *latest = validDocument(home); return nil })
	for _, err := range []error{readErr, updateErr} {
		var failure *StorageError
		if !errors.As(err, &failure) || failure.Operation != "read" || failure.Path != home || failure.Cause == nil || !errors.Is(err, failure.Cause) {
			t.Fatalf("error lost storage context: %v", err)
		}
	}
	files, err := os.ReadDir(filepath.Dir(home))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".robert-") {
			t.Fatalf("temporary file left behind: %s", file.Name())
		}
	}
}

func TestUpdatePreservesJSONSchemaAndPrivatePermissions(t *testing.T) {
	home := t.TempDir()
	store := NewJSONFileCatalog(home)
	doc := validDocument(home)
	if err := store.Update(func(latest *Document) error { *latest = doc; return nil }); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 3 || saved["version"] == nil || saved["installDir"] == nil || saved["repositories"] == nil {
		t.Fatalf("unexpected document schema: %s", data)
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(saved["repositories"], &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0]) != 4 || entries[0]["url"] == nil || entries[0]["path"] == nil || entries[0]["reference"] == nil || entries[0]["addedAt"] == nil {
		t.Fatalf("unexpected entry schema: %s", data)
	}
	var ref map[string]string
	if err := json.Unmarshal(entries[0]["reference"], &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != 2 || ref["type"] != "branch" || ref["value"] != "main" {
		t.Fatalf("unexpected reference schema: %v", ref)
	}
	info, err := os.Stat(store.Path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private file permissions: %v, %v", info, err)
	}
}
