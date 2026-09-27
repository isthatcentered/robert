package add

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

func TestAddRechecksLatestCatalogueAfterCloning(t *testing.T) {
	for _, tc := range []struct {
		name       string
		same       bool
		cleanupErr error
	}{
		{name: "different installations"},
		{name: "same installation", same: true},
		{name: "redundant cleanup fails silently", same: true, cleanupErr: errors.New("permission denied")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := catalog.NewJSONFileCatalog(t.TempDir())
			cloneStarted := make(chan struct{})
			resumeClone := make(chan struct{})
			t.Cleanup(func() { close(resumeClone) })
			dirs := &cleanupDirs{store: store, removeErr: tc.cleanupErr}
			first := Logic{Catalog: store, Git: pausedGit{cloneStarted, resumeClone}, Dirs: dirs}
			selection := Selection{URL: "https://example.com/first.git", Reference: &catalog.Reference{Type: "branch", Value: "main"}}
			firstDone := make(chan addOutcome, 1)
			go func() {
				result, err := first.Add(context.Background(), selection)
				firstDone <- addOutcome{result, err}
			}()
			select {
			case <-cloneStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("clone did not start")
			}

			// This must complete while the first clone is still paused.
			secondSelection := selection
			if !tc.same {
				secondSelection.URL = "https://example.com/second.git"
			}
			secondDone := make(chan addOutcome, 1)
			go func() {
				result, err := (Logic{Catalog: store, Git: successfulGit{}, Dirs: Directory{}}).Add(context.Background(), secondSelection)
				secondDone <- addOutcome{result, err}
			}()
			second := receiveAdd(t, secondDone)
			if second.err != nil {
				t.Fatal(second.err)
			}
			// Other catalogue fields must also survive the first command's stale snapshot.
			if err := store.Update(func(doc *catalog.Document) error { doc.Version = 9; return nil }); err != nil {
				t.Fatal(err)
			}
			resumeClone <- struct{}{}
			result := receiveAdd(t, firstDone)
			if result.err != nil {
				t.Fatal(result.err)
			}
			doc, err := store.Read()
			if err != nil || doc.Version != 9 {
				t.Fatalf("lost latest catalogue data: %+v, %v", doc, err)
			}
			if tc.same {
				if len(doc.Repositories) != 1 || result.result != second.result {
					t.Fatalf("duplicate did not return first saved data: %+v, %+v, %+v", result, second, doc)
				}
				if dirs.removed != dirs.created || dirs.removed == second.result.Path {
					t.Fatalf("wrong redundant cleanup: %+v", dirs)
				}
				if dirs.readErr != nil || len(dirs.snapshot.Repositories) != 1 {
					t.Fatalf("cleanup could not read saved catalogue: %+v, %v", dirs.snapshot, dirs.readErr)
				}
				if _, err := os.Stat(dirs.created); (tc.cleanupErr == nil && !os.IsNotExist(err)) || (tc.cleanupErr != nil && err != nil) {
					t.Fatalf("unexpected redundant checkout state: %v", err)
				}
			} else {
				if len(doc.Repositories) != 2 || dirs.removed != "" {
					t.Fatalf("lost concurrent addition: %+v, cleanup=%q", doc, dirs.removed)
				}
			}
			if result.result.Status != "added" {
				t.Fatalf("status = %q", result.result.Status)
			}
			if _, err := os.Stat(second.result.Path); err != nil {
				t.Fatalf("winning checkout removed: %v", err)
			}
		})
	}
}

func TestAddDoesNotDeleteCheckoutAfterCommittedUpdateError(t *testing.T) {
	home := t.TempDir()
	store := &failingStore{doc: catalog.Document{Version: 1, InstallDir: home}, saveErr: &catalog.StorageError{
		Operation: "unlock", Path: filepath.Join(home, ".robert"), LockPath: filepath.Join(home, ".robert.lock"), Committed: true, Cause: errors.New("unlock failed"),
	}}
	dirs := &recordingDirs{path: filepath.Join(home, "checkout")}
	_, err := (Logic{Catalog: store, Git: successfulGit{}, Dirs: dirs}).Add(context.Background(), Selection{URL: "https://example.com/repo.git", Reference: &catalog.Reference{Type: "branch", Value: "main"}})
	if err == nil || dirs.removed != "" {
		t.Fatalf("committed update cleanup=%q, error=%v", dirs.removed, err)
	}
	failure := problem.AsError(err)
	if failure.Context["committed"] != true || failure.Context["lockPath"] == "" {
		t.Fatalf("missing commit context: %+v", failure)
	}
}

type pausedGit struct {
	started chan<- struct{}
	resume  <-chan struct{}
}

func (pausedGit) DefaultBranch(context.Context, string) (string, error) { return "main", nil }
func (g pausedGit) Install(context.Context, string, catalog.Reference, string) error {
	close(g.started)
	<-g.resume
	return nil
}

type cleanupDirs struct {
	store              catalog.JSONFileCatalog
	created, removed   string
	removeErr, readErr error
	snapshot           catalog.Document
}

func (d *cleanupDirs) Create(root string) (string, error) {
	path, err := (Directory{}).Create(root)
	d.created = path
	return path, err
}
func (d *cleanupDirs) Remove(path string) error {
	d.removed = path
	// Re-entering the catalogue proves cleanup is outside the update lock.
	d.snapshot, d.readErr = d.store.Read()
	if d.removeErr != nil {
		return d.removeErr
	}
	return (Directory{}).Remove(path)
}

type addOutcome struct {
	result Result
	err    error
}

func receiveAdd(t *testing.T, done <-chan addOutcome) addOutcome {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(5 * time.Second):
		t.Fatal("add did not finish while clone or cleanup was outside the lock")
		return addOutcome{}
	}
}
