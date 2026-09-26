package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Entry struct {
	URL       string               `json:"url"`
	Path      string               `json:"path"`
	Reference repository.Reference `json:"reference"`
	AddedAt   string               `json:"addedAt"`
}

type Document struct {
	Version      int     `json:"version"`
	InstallDir   string  `json:"installDir"`
	Repositories []Entry `json:"repositories"`
}

// Validate checks saved fields, not installation uniqueness or checkout existence.
func (d Document) Validate() error {
	if d.Version != 1 {
		return fmt.Errorf("unsupported configuration version %d (expected 1)", d.Version)
	}
	if !filepath.IsAbs(d.InstallDir) || strings.ContainsRune(d.InstallDir, 0) {
		return fmt.Errorf("installDir must be a nonempty absolute path without NUL characters")
	}
	if d.Repositories == nil {
		return fmt.Errorf("repositories must be an array")
	}
	for i, entry := range d.Repositories {
		if err := entry.validate(); err != nil {
			return fmt.Errorf("repositories[%d] (%q): %w", i, entry.URL, err)
		}
	}
	return nil
}

func (e Entry) validate() error {
	normalized, err := repository.NormalizeURL(e.URL)
	if err != nil {
		return fmt.Errorf("url: %w", err)
	}
	if normalized != e.URL {
		return fmt.Errorf("url must be a full remote URL, not owner/repo shorthand")
	}
	if !filepath.IsAbs(e.Path) || strings.ContainsRune(e.Path, 0) {
		return fmt.Errorf("path must be a nonempty absolute checkout path without NUL characters")
	}
	if err := e.Reference.Validate(); err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, e.AddedAt); err != nil {
		return fmt.Errorf("addedAt must be an RFC3339 timestamp: %w", err)
	}
	return nil
}

func New(home string) Document {
	return Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []Entry{}}
}

type Store struct {
	Path string
	Home string
}

func (s Store) Load() (Document, bool, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return New(s.Home), false, nil
	}
	if err != nil {
		return Document{}, false, err
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return Document{}, true, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := doc.Validate(); err != nil {
		return Document{}, true, err
	}
	return doc, true, nil
}

func (s Store) Save(doc Document) error {
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("refusing to write invalid configuration: %w", err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	data = append(data, '\n')
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
	if err := os.Rename(tmp.Name(), s.Path); err != nil {
		return err
	}
	return nil
}
