package catalog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestConcurrentUpdatesPreserveEveryChange(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	const writers = 20
	start := make(chan struct{})
	done := make(chan error, writers)
	for i := range writers {
		go func() {
			<-start
			done <- store.Update(func(doc *Document) error {
				doc.Repositories = append(doc.Repositories, Entry{URL: RepositoryURL(fmt.Sprintf("https://example.com/repo-%d.git", i))})
				return nil
			})
		}()
	}
	close(start)
	for range writers {
		awaitOperation(t, done)
	}
	doc, err := store.Read()
	if err != nil || len(doc.Repositories) != writers {
		t.Fatalf("Read = %+v, %v; want %d entries", doc, err, writers)
	}
	seen := map[RepositoryURL]bool{}
	for _, entry := range doc.Repositories {
		if seen[entry.URL] {
			t.Fatalf("duplicate entry %q", entry.URL)
		}
		seen[entry.URL] = true
	}
}

func TestSharedReadersAndExclusiveUpdates(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	first, err := store.lock(true)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Unlock()
	second, err := store.lock(true)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Unlock()

	read := make(chan error, 1)
	go func() { _, err := store.Read(); read <- err }()
	awaitOperation(t, read) // A catalogue read shares the held reader locks.

	writer := make(chan error, 1)
	go func() { writer <- store.Update(func(doc *Document) error { doc.Version++; return nil }) }()
	assertBlocked(t, writer)
	if err := first.Unlock(); err != nil {
		t.Fatal(err)
	}
	assertBlocked(t, writer) // One reader must not release another reader's lock.
	if err := second.Unlock(); err != nil {
		t.Fatal(err)
	}
	awaitOperation(t, writer)
}

func TestUpdateExcludesReadersAndWritersOnSameInstance(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	first := make(chan error, 1)
	go func() {
		first <- store.Update(func(doc *Document) error {
			close(entered)
			<-release
			doc.Version++
			return nil
		})
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first update did not acquire the lock")
	}
	read := make(chan error, 1)
	go func() { _, err := store.Read(); read <- err }()
	write := make(chan error, 1)
	go func() { write <- store.Update(func(doc *Document) error { doc.Version++; return nil }) }()
	assertBlocked(t, read)
	assertBlocked(t, write)
	release <- struct{}{}
	awaitOperation(t, first)
	awaitOperation(t, read)
	awaitOperation(t, write)
	doc, err := store.Read()
	if err != nil || doc.Version != 3 {
		t.Fatalf("Read = %+v, %v; want both updates", doc, err)
	}
}

func TestUpdateAbortAndPanicReleaseLockWithoutSaving(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(fmt.Sprintf("panic=%t", panics), func(t *testing.T) {
			store := NewJSONFileCatalog(t.TempDir())
			if err := store.Update(func(doc *Document) error { doc.Version = 7; return nil }); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(store.Path)
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("abort")
			func() {
				if panics {
					defer func() {
						if recover() != failure {
							t.Error("callback panic was not propagated")
						}
					}()
				}
				err := store.Update(func(doc *Document) error {
					doc.Version = 99
					if panics {
						panic(failure)
					}
					return failure
				})
				if !errors.Is(err, failure) {
					t.Errorf("Update = %v; want callback error", err)
				}
			}()
			after, err := os.ReadFile(store.Path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("aborted update changed catalogue: %v", err)
			}
			done := make(chan error, 1)
			go func() { done <- store.Update(func(doc *Document) error { doc.Version++; return nil }) }()
			awaitOperation(t, done)
		})
	}
}

func TestFailedSavePreservesCatalogueAndReleasesLock(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	if err := store.Update(func(doc *Document) error { doc.Version = 7; return nil }); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Move the parent away while the update holds its already-open lock file
	// to force a save failure, even when tests run as root.
	moved := store.Home + "-moved"
	t.Cleanup(func() { _ = os.RemoveAll(moved) })
	err = store.Update(func(doc *Document) error {
		doc.Version++
		return os.Rename(store.Home, moved)
	})
	if renameErr := os.Rename(moved, store.Home); renameErr != nil {
		t.Fatal(renameErr)
	}
	var storage *StorageError
	if !errors.As(err, &storage) || storage.Operation != "write" || storage.Committed {
		t.Fatalf("expected uncommitted save failure, got %v", err)
	}
	after, err := os.ReadFile(store.Path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("failed save changed catalogue: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- store.Update(func(doc *Document) error { doc.Version++; return nil }) }()
	awaitOperation(t, done)
}

func TestLockFilePersistsAcrossReplacement(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	if _, err := store.Read(); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(store.Path + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode().Perm() != 0600 {
		t.Fatalf("lock permissions = %v", before.Mode())
	}
	for range 3 {
		if err := store.Update(func(doc *Document) error { doc.Version++; return nil }); err != nil {
			t.Fatal(err)
		}
		after, err := os.Stat(store.Path + ".lock")
		if err != nil || !os.SameFile(before, after) {
			t.Fatalf("lock file replaced: %v", err)
		}
	}
}

func TestIndependentProcessesPreserveUpdates(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	var children []*catalogProcess
	for range 6 {
		children = append(children, startCatalogProcess(t, store.Path, "update"))
	}
	for _, child := range children {
		child.start(t)
	}
	for _, child := range children {
		awaitOperation(t, child.done)
	}
	doc, err := store.Read()
	if err != nil || doc.Version != 61 {
		t.Fatalf("Read = %+v, %v; want version 61", doc, err)
	}
}

func TestProcessReadWaitsForUpdate(t *testing.T) {
	store := NewJSONFileCatalog(t.TempDir())
	lock, err := store.lock(false)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	child := startCatalogProcess(t, store.Path, "read")
	child.start(t)
	assertBlocked(t, child.done)
	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	awaitOperation(t, child.done)
}

// TestCatalogProcess is run in a separate process by the tests above.
func TestCatalogProcess(t *testing.T) {
	path := os.Getenv("ROBERT_TEST_CATALOG_PATH")
	if path == "" {
		return
	}
	store := JSONFileCatalog{Path: path, Home: filepath.Dir(path)}
	fmt.Println("ready")
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatal(err)
	}
	switch os.Getenv("ROBERT_TEST_CATALOG_OPERATION") {
	case "update":
		for range 10 {
			if err := store.Update(func(doc *Document) error { doc.Version++; return nil }); err != nil {
				t.Fatal(err)
			}
		}
	case "read":
		if _, err := store.Read(); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("unknown child operation")
	}
}

type catalogProcess struct {
	stdin io.WriteCloser
	done  chan error
}

func startCatalogProcess(t *testing.T, path, operation string) *catalogProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCatalogProcess$")
	cmd.Env = append(os.Environ(), "ROBERT_TEST_CATALOG_PATH="+path, "ROBERT_TEST_CATALOG_OPERATION="+operation, "GORACE=atexit_sleep_ms=0")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stdin.Close() })
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(stdout)
	if line, err := reader.ReadString('\n'); err != nil || line != "ready\n" {
		_ = cmd.Wait()
		t.Fatalf("child not ready: %q, %v: %s", line, err, stderr.String())
	}
	child := &catalogProcess{stdin: stdin, done: make(chan error, 1)}
	go func() {
		output, readErr := io.ReadAll(reader)
		err := cmd.Wait()
		if err != nil {
			err = fmt.Errorf("%w: %s%s", err, output, stderr.String())
		}
		child.done <- errors.Join(err, readErr)
	}()
	return child
}

func (p *catalogProcess) start(t *testing.T) {
	t.Helper()
	if err := p.stdin.Close(); err != nil {
		t.Fatal(err)
	}
}

func awaitOperation(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("catalogue operation did not finish")
	}
}

func assertBlocked(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("operation completed while an incompatible lock was held: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}
