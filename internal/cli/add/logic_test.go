package add

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

func TestSaveFailureRemovesCheckoutAndReportsContext(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "install", "new-checkout")
	store := &failingStore{doc: catalog.Document{Version: 1, InstallDir: filepath.Join(home, ".agents", "robert"), Repositories: []catalog.Entry{}}, saveErr: &catalog.StorageError{Operation: "write", Path: filepath.Join(home, ".robert"), Cause: errors.New("disk is full")}}
	dirs := &recordingDirs{path: path, removeErr: errors.New("permission denied")}
	logic := Logic{Catalog: store, Git: successfulGit{}, Dirs: dirs, Now: func() time.Time { return time.Unix(0, 0) }}
	_, err := logic.Add(context.Background(), Selection{URL: "https://example.com/demo.git", Reference: &catalog.Reference{Type: "branch", Value: "main"}})
	if err == nil {
		t.Fatal("expected save error")
	}
	failure := problem.AsError(err)
	if failure.Message != "failed to save configuration after cloning repository" {
		t.Fatalf("message = %q", failure.Message)
	}
	if failure.Context["cause"] != "disk is full" || failure.Context["cleanupError"] != "permission denied" || failure.Context["path"] != path {
		t.Fatalf("incomplete error context: %v", failure.Context)
	}
	if failure.Context["configPath"] != filepath.Join(home, ".robert") || failure.Context["hint"] == "" {
		t.Fatalf("missing storage error context: %v", failure.Context)
	}
	if dirs.removed != path {
		t.Fatalf("cleanup path = %q, want %q", dirs.removed, path)
	}
}

type failingStore struct {
	doc     catalog.Document
	saveErr error
}

func (s *failingStore) Read() (catalog.Document, error) { return s.doc, nil }
func (s *failingStore) Update(change func(*catalog.Document) error) error {
	doc := s.doc
	if err := change(&doc); err != nil {
		return err
	}
	return s.saveErr
}

type successfulGit struct{}

func (successfulGit) DefaultBranch(context.Context, string) (string, error)            { return "main", nil }
func (successfulGit) Install(context.Context, string, catalog.Reference, string) error { return nil }

type recordingDirs struct {
	path      string
	removed   string
	removeErr error
}

func (d *recordingDirs) Create(string) (string, error) { return d.path, nil }
func (d *recordingDirs) Remove(path string) error      { d.removed = path; return d.removeErr }
