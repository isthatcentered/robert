package remove

import (
	"errors"
	"path/filepath"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Logic struct {
	Catalog catalog.Catalog
	Dirs    CheckoutDirs
}

type Result struct {
	Status    string               `json:"status"`
	URL       string               `json:"url"`
	Path      string               `json:"path"`
	Reference repository.Reference `json:"reference"`
	AddedAt   string               `json:"addedAt"`
}

func (l Logic) Remove(selection Selection) (*Result, error) {
	var entry catalog.Entry
	changed := false
	err := l.Catalog.Update(func(doc *catalog.Document) error {
		index := -1
		var references []repository.Reference
		for i, entry := range doc.Repositories {
			if entry.URL != selection.URL {
				continue
			}
			if selection.Reference != nil && !repository.Reference(entry.Reference).Equal(*selection.Reference) {
				continue
			}
			if index == -1 {
				index = i
			}
			references = append(references, repository.Reference(entry.Reference))
		}
		if len(references) > 1 {
			context := map[string]any{"url": selection.URL, "matchingReferences": references}
			if selection.Reference != nil {
				return problem.New("multiple identical installations match; remove duplicate entries from configuration", context)
			}
			return problem.New("multiple installations match this repository; specify --branch, --tag, or --commit", context)
		}
		if index == -1 {
			context := map[string]any{"url": selection.URL, "hint": "use list to see saved installations, or add this repository before removing it"}
			if selection.Reference != nil {
				context["reference"] = selection.Reference
			}
			return problem.New("repository installation not found in configuration", context)
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
	_ = l.Dirs.Remove(entry.Path)
	return &Result{Status: "removed", URL: entry.URL, Path: entry.Path, Reference: repository.Reference(entry.Reference), AddedAt: entry.AddedAt}, nil
}
