package remove

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/isthatcentered/robert/internal/cli/config"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

func TestRemoveSucceedsAfterSaveWhenCheckoutDeletionFails(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "install", "checkout")
	doc := config.New(home)
	doc.Repositories = []config.Entry{{URL: "https://example.com/demo.git", Path: path, Reference: repository.Reference{Type: "branch", Value: "main"}, AddedAt: "2026-09-26T12:00:00Z"}}
	store := &recordingStore{doc: doc}
	dirs := &failingDirs{store: store}
	logic := Logic{Config: store, Dirs: dirs, ConfigPath: filepath.Join(home, ".robert")}
	result, err := logic.Remove(Selection{URL: "https://example.com/demo.git"})
	if err != nil {
		t.Fatal(err)
	}
	if string(result["status"]) != `"removed"` {
		t.Fatalf("status = %s", result["status"])
	}
	if len(store.doc.Repositories) != 0 {
		t.Fatalf("entry was retained: %v", store.doc.Repositories)
	}
	if !dirs.called || !dirs.sawSaved {
		t.Fatal("directory deletion did not happen after configuration save")
	}
	if _, exists := result["cleanupError"]; exists {
		t.Fatal("cleanup failure appeared in successful result")
	}
}

func TestRemoveKeepsEntryAndDirectoryWhenSaveFails(t *testing.T) {
	home := t.TempDir()
	doc := config.New(home)
	doc.Repositories = []config.Entry{{URL: "https://example.com/demo.git", Path: filepath.Join(home, "checkout"), Reference: repository.Reference{Type: "branch", Value: "main"}}}
	store := &recordingStore{doc: doc, saveErr: errors.New("disk is full")}
	dirs := &failingDirs{store: store}
	logic := Logic{Config: store, Dirs: dirs, ConfigPath: filepath.Join(home, ".robert")}
	_, err := logic.Remove(Selection{URL: "https://example.com/demo.git"})
	if err == nil {
		t.Fatal("expected save error")
	}
	if len(store.doc.Repositories) != 1 {
		t.Fatal("entry changed after failed save")
	}
	if dirs.called {
		t.Fatal("checkout deletion started after failed save")
	}
}

type recordingStore struct {
	doc     config.Document
	saved   bool
	saveErr error
}

func (s *recordingStore) Load() (config.Document, bool, error) { return s.doc, true, nil }
func (s *recordingStore) Save(doc config.Document) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.doc = doc
	s.saved = true
	return nil
}

type failingDirs struct {
	store    *recordingStore
	called   bool
	sawSaved bool
}

func (d *failingDirs) Remove(string) error {
	d.called = true
	d.sawSaved = d.store.saved
	return errors.New("permission denied")
}
