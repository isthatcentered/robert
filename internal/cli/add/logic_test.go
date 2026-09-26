package add

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/isthatcentered/robert/internal/cli/arguments"
	"github.com/isthatcentered/robert/internal/cli/config"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

func TestSaveFailureRemovesCheckoutAndReportsContext(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "install", "new-checkout")
	store := &failingStore{doc: config.New(home), saveErr: errors.New("disk is full")}
	dirs := &recordingDirs{path: path, removeErr: errors.New("permission denied")}
	logic := Logic{Config: store, Git: successfulGit{}, Dirs: dirs, ConfigPath: filepath.Join(home, ".robert"), Now: func() time.Time { return time.Unix(0, 0) }}
	_, err := logic.Add(context.Background(), arguments.Selection{URL: "https://example.com/demo.git", Reference: &repository.Reference{Type: "branch", Value: "main"}})
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
	if dirs.removed != path {
		t.Fatalf("cleanup path = %q, want %q", dirs.removed, path)
	}
}

type failingStore struct {
	doc     config.Document
	saveErr error
}

func (s *failingStore) Load() (config.Document, bool, error) { return s.doc, false, nil }
func (s *failingStore) Save(config.Document) error           { return s.saveErr }

type successfulGit struct{}

func (successfulGit) DefaultBranch(context.Context, string) (string, error)               { return "main", nil }
func (successfulGit) Install(context.Context, string, repository.Reference, string) error { return nil }

type recordingDirs struct {
	path      string
	removed   string
	removeErr error
}

func (d *recordingDirs) Create(string) (string, error) { return d.path, nil }
func (d *recordingDirs) Remove(path string) error      { d.removed = path; return d.removeErr }
