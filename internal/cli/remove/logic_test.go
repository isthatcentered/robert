package remove

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

func TestRemoveSucceedsAfterSaveWhenCheckoutDeletionFails(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "install", "checkout")
	doc := catalog.Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []catalog.Entry{}}
	doc.Repositories = []catalog.Entry{{URL: "https://example.com/demo.git", Path: path, Reference: catalog.Reference{Type: "branch", Value: "main"}, AddedAt: "2026-09-26T12:00:00Z"}}
	store := &recordingStore{doc: doc}
	dirs := &failingDirs{store: store}
	logic := Logic{Catalog: store, Dirs: dirs}
	result, err := logic.Remove(Selection{URL: "https://example.com/demo.git"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "removed" {
		t.Fatalf("status = %s", result.Status)
	}
	if got := FormatWarning(result); got != "warning: could not delete checkout: permission denied\n  Checkout: "+path+"\n" {
		t.Fatalf("warning = %q", got)
	}
	if len(store.doc.Repositories) != 0 {
		t.Fatalf("entry was retained: %v", store.doc.Repositories)
	}
	if !dirs.called || !dirs.sawSaved {
		t.Fatal("directory deletion did not happen after configuration save")
	}
}

func TestRemoveKeepsEntryAndDirectoryWhenSaveFails(t *testing.T) {
	home := t.TempDir()
	doc := catalog.Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []catalog.Entry{}}
	doc.Repositories = []catalog.Entry{{URL: "https://example.com/demo.git", Path: filepath.Join(home, "checkout"), Reference: catalog.Reference{Type: "branch", Value: "main"}}}
	store := &recordingStore{doc: doc, saveErr: &catalog.StorageError{Operation: "write", Path: filepath.Join(home, ".robert"), Cause: errors.New("disk is full")}}
	dirs := &failingDirs{store: store}
	logic := Logic{Catalog: store, Dirs: dirs}
	_, err := logic.Remove(Selection{URL: "https://example.com/demo.git"})
	if err == nil {
		t.Fatal("expected save error")
	}
	failure := problem.AsError(err)
	if failure.Context["configPath"] != filepath.Join(home, ".robert") || failure.Context["cause"] != "disk is full" || failure.Context["hint"] == "" {
		t.Fatalf("missing storage error context: %v", failure.Context)
	}
	if len(store.doc.Repositories) != 1 {
		t.Fatal("entry changed after failed save")
	}
	if dirs.called {
		t.Fatal("checkout deletion started after failed save")
	}
}

type recordingStore struct {
	doc     catalog.Document
	saved   bool
	saveErr error
}

func (s *recordingStore) Read() (catalog.Document, error) { return s.doc, nil }
func (s *recordingStore) Update(change func(*catalog.Document) error) error {
	doc := s.doc
	if err := change(&doc); err != nil {
		return err
	}
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
