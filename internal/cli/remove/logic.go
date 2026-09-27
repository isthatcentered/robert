package remove

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

type Logic struct {
	Catalog catalog.Catalog
	Dirs    CheckoutDirs
}

type Result struct {
	Status       string
	URL          string
	Path         string
	Reference    catalog.Reference
	AddedAt      string
	CleanupError error
}

func Format(result *Result) string {
	return fmt.Sprintf("Removed from catalogue: %s\n  Reference: %s %s\n  Checkout:  %s\n  Added:     %s\n", result.URL, result.Reference.Type, result.Reference.Value, result.Path, result.AddedAt)
}

func FormatWarning(result *Result) string {
	if result.CleanupError == nil {
		return ""
	}
	return fmt.Sprintf("warning: could not delete checkout: %s\n  Checkout: %s\n", result.CleanupError, result.Path)
}

func (l Logic) Remove(selection Selection) (*Result, error) {
	var entry catalog.Entry
	changed := false
	err := l.Catalog.Update(func(doc *catalog.Document) error {
		index := -1
		var references []catalog.Reference
		for i, entry := range doc.Repositories {
			if entry.URL != selection.URL {
				continue
			}
			if selection.Reference != nil && !entry.Reference.Equal(*selection.Reference) {
				continue
			}
			if index == -1 {
				index = i
			}
			references = append(references, entry.Reference)
		}
		if len(references) > 1 {
			context := map[string]any{"url": selection.URL, "matchingReferences": references}
			if selection.Reference != nil {
				context["hint"] = "remove duplicate entries from the catalogue"
				return problem.New("multiple identical checkouts match this repository", context)
			}
			context["hint"] = "specify --branch, --tag, or --commit"
			return problem.New("multiple saved checkouts match this repository", context)
		}
		if index == -1 {
			context := map[string]any{"url": selection.URL, "hint": "run \"robert list\" to see saved repositories, or add this repository before removing it"}
			if selection.Reference != nil {
				context["reference"] = selection.Reference
			}
			return problem.New("repository checkout not found in catalogue", context)
		}
		selected := doc.Repositories[index]
		if selected.Path == "" || !filepath.IsAbs(selected.Path) {
			return problem.New("saved checkout path must be a nonempty absolute path", map[string]any{"url": selected.URL, "path": selected.Path, "hint": "correct the saved checkout path before retrying removal"})
		}
		updated := make([]catalog.Entry, 0, len(doc.Repositories)-1)
		updated = append(updated, doc.Repositories[:index]...)
		updated = append(updated, doc.Repositories[index+1:]...)
		doc.Repositories = updated
		entry = selected
		changed = true
		return nil
	})
	if err != nil {
		if !changed {
			return nil, err
		}
		var storage *catalog.StorageError
		if errors.As(err, &storage) && storage.Committed {
			return nil, problem.Wrap("configuration saved but failed to finish catalogue update", err, map[string]any{"url": entry.URL, "checkoutPath": entry.Path})
		}
		return nil, problem.Wrap("failed to save configuration before removing checkout", err, map[string]any{"url": entry.URL, "checkoutPath": entry.Path})
	}
	cleanupErr := l.Dirs.Remove(entry.Path)
	return &Result{Status: "removed", URL: entry.URL, Path: entry.Path, Reference: entry.Reference, AddedAt: entry.AddedAt, CleanupError: cleanupErr}, nil
}
