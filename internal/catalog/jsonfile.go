package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
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
	URL       string    `json:"url"`
	Path      string    `json:"path"`
	Reference Reference `json:"reference"`
	AddedAt   string    `json:"addedAt"`
}

func (s JSONFileCatalog) Read() (doc Document, err error) {
	lock, err := s.lock(true)
	if err != nil {
		return Document{}, err
	}
	defer func() { err = errors.Join(err, s.unlock(lock, false)) }()
	return s.read()
}

func (s JSONFileCatalog) Update(change func(*Document) error) (err error) {
	lock, err := s.lock(false)
	if err != nil {
		return err
	}
	committed := false
	defer func() { err = errors.Join(err, s.unlock(lock, committed)) }()
	doc, err := s.read()
	if err != nil {
		return err
	}
	if err := change(&doc); err != nil {
		return err
	}
	if err := s.save(doc); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s JSONFileCatalog) lock(shared bool) (*flock.Flock, error) {
	// Each operation owns a separate handle. Reusing a Flock would allow a
	// second acquisition to short-circuit and an early unlock to release it.
	// The companion file survives replacement of the JSON file by rename.
	lock := flock.New(s.Path + ".lock")
	var err error
	if shared {
		err = lock.RLock()
	} else {
		err = lock.Lock()
	}
	if err != nil {
		return nil, &StorageError{Operation: "lock", Path: s.Path, LockPath: lock.Path(), Cause: err}
	}
	return lock, nil
}

func (s JSONFileCatalog) unlock(lock *flock.Flock, committed bool) error {
	if err := lock.Unlock(); err != nil {
		return &StorageError{Operation: "unlock", Path: s.Path, LockPath: lock.Path(), Committed: committed, Cause: err}
	}
	return nil
}

func (s JSONFileCatalog) read() (Document, error) {
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
		doc.Repositories[i] = Entry{URL: entry.URL, Path: entry.Path, Reference: entry.Reference, AddedAt: entry.AddedAt}
	}
	return doc, nil
}

func (s JSONFileCatalog) save(doc Document) error {
	saved := jsonDocument{Version: doc.Version, InstallDir: doc.InstallDir}
	if doc.Repositories != nil {
		saved.Repositories = make([]jsonEntry, len(doc.Repositories))
	}
	for i, entry := range doc.Repositories {
		saved.Repositories[i] = jsonEntry{URL: entry.URL, Path: entry.Path, Reference: entry.Reference, AddedAt: entry.AddedAt}
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
