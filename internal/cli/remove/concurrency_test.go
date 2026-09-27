package remove

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

func TestRemoveUnlocksBeforeDeletionAndKeepsConcurrentReplacement(t *testing.T) {
	home := t.TempDir()
	store := catalog.NewJSONFileCatalog(home)
	oldPath, newPath := filepath.Join(home, "old"), filepath.Join(home, "new")
	for _, path := range []string{oldPath, newPath} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	entry := catalog.Entry{URL: "https://example.com/repo.git", Path: oldPath, Reference: catalog.Reference{Type: "branch", Value: "main"}}
	if err := store.Update(func(doc *catalog.Document) error { doc.Repositories = append(doc.Repositories, entry); return nil }); err != nil {
		t.Fatal(err)
	}
	deleting := make(chan string, 1)
	resume := make(chan struct{})
	t.Cleanup(func() { close(resume) })
	logic := Logic{Catalog: store, Dirs: pausedDeletion{deleting, resume}}
	firstDone := make(chan removeOutcome, 1)
	go func() {
		result, err := logic.Remove(Selection{URL: entry.URL})
		firstDone <- removeOutcome{result, err}
	}()
	select {
	case path := <-deleting:
		if path != oldPath {
			t.Fatalf("deleting %q instead of %q", path, oldPath)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("checkout deletion did not start")
	}

	// A second remove observes the saved removal while deletion is still paused.
	secondDone := make(chan removeOutcome, 1)
	go func() {
		result, err := logic.Remove(Selection{URL: entry.URL})
		secondDone <- removeOutcome{result, err}
	}()
	second := receiveRemove(t, secondDone)
	if second.err == nil || problem.AsError(second.err).Message != "repository checkout not found in catalogue" {
		t.Fatalf("second remove = %+v", second)
	}

	entry.Path = newPath
	if err := store.Update(func(doc *catalog.Document) error { doc.Repositories = append(doc.Repositories, entry); return nil }); err != nil {
		t.Fatal(err)
	}
	resume <- struct{}{}
	first := receiveRemove(t, firstDone)
	if first.err != nil || first.result.Path != oldPath {
		t.Fatalf("first remove = %+v", first)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old checkout remains: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("replacement checkout removed: %v", err)
	}
	doc, err := store.Read()
	if err != nil || len(doc.Repositories) != 1 || doc.Repositories[0].Path != newPath {
		t.Fatalf("replacement entry lost: %+v, %v", doc, err)
	}
}

type pausedDeletion struct {
	deleting chan<- string
	resume   <-chan struct{}
}

func (d pausedDeletion) Remove(path string) error {
	d.deleting <- path
	<-d.resume
	return os.RemoveAll(path)
}

type removeOutcome struct {
	result *Result
	err    error
}

func receiveRemove(t *testing.T, done <-chan removeOutcome) removeOutcome {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(5 * time.Second):
		t.Fatal("remove did not finish while deletion was outside the lock")
		return removeOutcome{}
	}
}
