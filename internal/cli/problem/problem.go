package problem

import "errors"

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
	return &Error{Message: err.Error()}
}
