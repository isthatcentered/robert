package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Entry struct {
	URL       string
	Path      string
	Reference repository.Reference
	AddedAt   string
	fields    map[string]json.RawMessage
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("repository entry must be an object")
	}
	var known struct {
		URL       string               `json:"url"`
		Path      string               `json:"path"`
		Reference repository.Reference `json:"reference"`
		AddedAt   string               `json:"addedAt"`
	}
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*e = Entry{URL: known.URL, Path: known.Path, Reference: known.Reference, AddedAt: known.AddedAt, fields: fields}
	return nil
}

func (e Entry) MarshalJSON() ([]byte, error) {
	fields, err := e.jsonFields()
	if err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

func (e Entry) jsonFields() (map[string]json.RawMessage, error) {
	fields := clone(e.fields)
	if err := set(fields, "url", e.URL); err != nil {
		return nil, err
	}
	if err := set(fields, "path", e.Path); err != nil {
		return nil, err
	}
	if err := set(fields, "reference", e.Reference); err != nil {
		return nil, err
	}
	if err := set(fields, "addedAt", e.AddedAt); err != nil {
		return nil, err
	}
	return fields, nil
}

func (e Entry) Result(status string) (map[string]json.RawMessage, error) {
	fields, err := e.jsonFields()
	if err != nil {
		return nil, err
	}
	if err := set(fields, "status", status); err != nil {
		return nil, err
	}
	return fields, nil
}

type Document struct {
	Version      int
	InstallDir   string
	Repositories []Entry
	fields       map[string]json.RawMessage
}

func (d *Document) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("configuration must be an object")
	}
	var known struct {
		Version      int     `json:"version"`
		InstallDir   string  `json:"installDir"`
		Repositories []Entry `json:"repositories"`
	}
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	if known.Repositories == nil {
		return fmt.Errorf("configuration repositories must be an array")
	}
	*d = Document{Version: known.Version, InstallDir: known.InstallDir, Repositories: known.Repositories, fields: fields}
	return nil
}

func (d Document) MarshalJSON() ([]byte, error) {
	fields := clone(d.fields)
	if err := set(fields, "version", d.Version); err != nil {
		return nil, err
	}
	if err := set(fields, "installDir", d.InstallDir); err != nil {
		return nil, err
	}
	if d.Repositories == nil {
		d.Repositories = []Entry{}
	}
	if err := set(fields, "repositories", d.Repositories); err != nil {
		return nil, err
	}
	return json.Marshal(fields)
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
	if doc.Version != 1 {
		return Document{}, true, fmt.Errorf("unsupported configuration version %d (expected 1)", doc.Version)
	}
	if !filepath.IsAbs(doc.InstallDir) {
		return Document{}, true, fmt.Errorf("installDir must be an absolute path")
	}
	return doc, true, nil
}

func (s Store) Save(doc Document) error {
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

func clone(input map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(input)+4)
	for key, value := range input {
		result[key] = value
	}
	return result
}

func set(fields map[string]json.RawMessage, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fields[key] = encoded
	return nil
}
