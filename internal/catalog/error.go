package catalog

import "fmt"

// StorageError retains the storage location and underlying failure for callers.
type StorageError struct {
	Operation string
	Path      string
	Cause     error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("failed to %s configuration %q: %v", e.Operation, e.Path, e.Cause)
}

func (e *StorageError) Unwrap() error { return e.Cause }
