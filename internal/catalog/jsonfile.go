package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSONFileCatalog owns the on-disk format and configuration defaults.
type JSONFileCatalog struct {
	Path string
	Home string
}

var _ Catalog = JSONFileCatalog{}

func NewJSONFileCatalog(home string) JSONFileCatalog {
	return JSONFileCatalog{Path: filepath.Join(home, ".robert"), Home: home}
}

type jsonDocument struct {
	Version      int         `json:"version"`
	InstallDir   string      `json:"installDir"`
	Repositories []jsonEntry `json:"repositories"`
}

type jsonEntry struct {
	URL       string        `json:"url"`
	Path      string        `json:"path"`
	Reference jsonReference `json:"reference"`
	AddedAt   string        `json:"addedAt"`
}

type jsonReference struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (s JSONFileCatalog) Read() (Document, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return Document{Version: 1, InstallDir: filepath.Join(s.Home, ".agents", "robert"), Repositories: []Entry{}}, nil
	}
	if err != nil {
		return Document{}, &StorageError{Operation: "read", Path: s.Path, Cause: err}
	}
	var saved jsonDocument
	if err := json.Unmarshal(data, &saved); err != nil {
		return Document{}, &StorageError{Operation: "read", Path: s.Path, Cause: fmt.Errorf("invalid JSON: %w", err)}
	}
	doc := Document{Version: saved.Version, InstallDir: saved.InstallDir}
	if saved.Repositories != nil {
		doc.Repositories = make([]Entry, len(saved.Repositories))
	}
	for i, entry := range saved.Repositories {
		doc.Repositories[i] = Entry{URL: entry.URL, Path: entry.Path, Reference: Reference(entry.Reference), AddedAt: entry.AddedAt}
	}
	return doc, nil
}

func (s JSONFileCatalog) Write(doc Document) error {
	saved := jsonDocument{Version: doc.Version, InstallDir: doc.InstallDir}
	if doc.Repositories != nil {
		saved.Repositories = make([]jsonEntry, len(doc.Repositories))
	}
	for i, entry := range doc.Repositories {
		saved.Repositories[i] = jsonEntry{URL: entry.URL, Path: entry.Path, Reference: jsonReference(entry.Reference), AddedAt: entry.AddedAt}
	}
	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return &StorageError{Operation: "write", Path: s.Path, Cause: err}
	}
	data = append(data, '\n')
	if err := s.write(data); err != nil {
		return &StorageError{Operation: "write", Path: s.Path, Cause: err}
	}
	return nil
}

func (s JSONFileCatalog) write(data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".robert-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}
