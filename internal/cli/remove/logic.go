package remove

import (
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
	doc, err := l.Catalog.Read()
	if err != nil {
		return nil, err
	}
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
			return nil, problem.New("multiple identical installations match; remove duplicate entries from configuration", context)
		}
		return nil, problem.New("multiple installations match this repository; specify --branch, --tag, or --commit", context)
	}
	if index == -1 {
		context := map[string]any{"url": selection.URL, "hint": "use list to see saved installations, or add this repository before removing it"}
		if selection.Reference != nil {
			context["reference"] = selection.Reference
		}
		return nil, problem.New("repository installation not found in configuration", context)
	}
	entry := doc.Repositories[index]
	if entry.Path == "" || !filepath.IsAbs(entry.Path) {
		return nil, problem.New("saved checkout path must be a nonempty absolute path", map[string]any{"url": entry.URL, "path": entry.Path, "hint": "correct the saved checkout path before retrying removal"})
	}
	updated := make([]catalog.Entry, 0, len(doc.Repositories)-1)
	updated = append(updated, doc.Repositories[:index]...)
	updated = append(updated, doc.Repositories[index+1:]...)
	doc.Repositories = updated
	if err := l.Catalog.Write(doc); err != nil {
		return nil, problem.Wrap("failed to save configuration before removing checkout", err, map[string]any{"url": entry.URL, "checkoutPath": entry.Path})
	}
	_ = l.Dirs.Remove(entry.Path)
	return &Result{Status: "removed", URL: entry.URL, Path: entry.Path, Reference: repository.Reference(entry.Reference), AddedAt: entry.AddedAt}, nil
}
