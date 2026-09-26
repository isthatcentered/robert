package remove

import (
	"path/filepath"

	"github.com/isthatcentered/robert/internal/cli/config"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Logic struct {
	Config     ConfigStore
	Dirs       CheckoutDirs
	ConfigPath string
}

type Result struct {
	Status string `json:"status"`
	config.Entry
}

func (l Logic) Remove(selection Selection) (*Result, error) {
	doc, exists, err := l.Config.Load()
	if err != nil {
		return nil, problem.New("failed to read configuration", map[string]any{"path": l.ConfigPath, "cause": err.Error(), "hint": "check file permissions and correct the reported configuration fields before retrying"})
	}
	if !exists {
		return nil, problem.New("configuration does not exist", map[string]any{"path": l.ConfigPath, "url": selection.URL, "hint": "add a repository before removing one"})
	}
	index := -1
	var references []repository.Reference
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
			return nil, problem.New("multiple identical installations match; remove duplicate entries from configuration", context)
		}
		return nil, problem.New("multiple installations match this repository; specify --branch, --tag, or --commit", context)
	}
	if index == -1 {
		context := map[string]any{"url": selection.URL}
		if selection.Reference != nil {
			context["reference"] = selection.Reference
		}
		return nil, problem.New("repository installation not found in configuration", context)
	}
	entry := doc.Repositories[index]
	if entry.Path == "" || !filepath.IsAbs(entry.Path) {
		return nil, problem.New("saved checkout path must be a nonempty absolute path", map[string]any{"url": entry.URL, "path": entry.Path, "configPath": l.ConfigPath})
	}
	updated := make([]config.Entry, 0, len(doc.Repositories)-1)
	updated = append(updated, doc.Repositories[:index]...)
	updated = append(updated, doc.Repositories[index+1:]...)
	doc.Repositories = updated
	if err := l.Config.Save(doc); err != nil {
		return nil, problem.New("failed to save configuration before removing checkout", map[string]any{"path": l.ConfigPath, "url": entry.URL, "checkoutPath": entry.Path, "cause": err.Error()})
	}
	_ = l.Dirs.Remove(entry.Path)
	return &Result{Status: "removed", Entry: entry}, nil
}
