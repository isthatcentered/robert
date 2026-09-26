package problem

import (
	"errors"

	"github.com/isthatcentered/robert/internal/catalog"
)

// Error is the JSON error returned by the CLI.
type Error struct {
	Message string         `json:"error"`
	Context map[string]any `json:"context,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func New(message string, context map[string]any) error {
	return &Error{Message: message, Context: context}
}

func AsError(err error) *Error {
	var result *Error
	if errors.As(err, &result) {
		return result
	}
	var storage *catalog.StorageError
	if errors.As(err, &storage) {
		pathKey := "path"
		hint := "check file permissions and correct the JSON syntax before retrying"
		if storage.Operation == "write" {
			pathKey = "configPath"
			hint = "check the configuration directory exists, is writable, and has free space before retrying"
		}
		return &Error{Message: "failed to " + storage.Operation + " configuration", Context: map[string]any{
			pathKey: storage.Path, "cause": storage.Cause.Error(), "hint": hint,
		}}
	}
	return &Error{Message: err.Error()}
}

// Wrap adds workflow details while retaining contextual errors from adapters.
func Wrap(message string, err error, context map[string]any) error {
	failure := AsError(err)
	merged := map[string]any{"cause": err.Error()}
	for key, value := range failure.Context {
		merged[key] = value
	}
	for key, value := range context {
		merged[key] = value
	}
	return New(message, merged)
}
