package catalog

import "fmt"

// StorageError retains the storage location and underlying failure for callers.
type StorageError struct {
	Operation string
	Path      string
	LockPath  string
	// Committed means the update was saved before a subsequent failure.
	// Callers must not roll back external resources referenced by that update.
	Committed bool
	Cause     error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("failed to %s configuration %q: %v", e.Operation, e.Path, e.Cause)
}

func (e *StorageError) Unwrap() error { return e.Cause }
