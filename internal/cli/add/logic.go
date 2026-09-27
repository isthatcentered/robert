package add

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Logic struct {
	Catalog catalog.Catalog
	Git     Git
	Dirs    CheckoutDirs
	Now     func() time.Time
}

type Result struct {
	Status    string
	URL       string
	Reference repository.Reference
	Path      string
}

func Format(result Result) string {
	return fmt.Sprintf("Repository: %s\n  Reference: %s %s\n  Checkout:  %s\n", result.URL, result.Reference.Type, result.Reference.Value, result.Path)
}

func (l Logic) Add(ctx context.Context, selection Selection) (Result, error) {
	doc, err := l.Catalog.Read()
	if err != nil {
		return Result{}, err
	}
	ref := selection.Reference
	if ref == nil {
		branch, err := l.Git.DefaultBranch(ctx, selection.URL)
		if err != nil {
			return Result{}, problem.New("failed to resolve repository default branch", gitContext(selection.URL, "", err))
		}
		ref = &repository.Reference{Type: "branch", Value: branch}
	}
	for _, entry := range doc.Repositories {
		if entry.URL == selection.URL && repository.Reference(entry.Reference).Equal(*ref) {
			return addedResult(entry), nil
		}
	}
	path, err := l.Dirs.Create(doc.InstallDir)
	if err != nil {
		return Result{}, problem.New("failed to create repository checkout directory", map[string]any{"url": selection.URL, "installDir": doc.InstallDir, "cause": err.Error()})
	}
	if err := l.Git.Install(ctx, selection.URL, *ref, path); err != nil {
		context := gitContext(selection.URL, path, err)
		if cleanupErr := l.Dirs.Remove(path); cleanupErr != nil {
			context["cleanupError"] = cleanupErr.Error()
		}
		return Result{}, problem.New("failed to clone repository", context)
	}
	now := time.Now
	if l.Now != nil {
		now = l.Now
	}
	entry := catalog.Entry{URL: selection.URL, Path: path, Reference: catalog.Reference(*ref), AddedAt: now().UTC().Format(time.RFC3339Nano)}
	redundant := false
	err = l.Catalog.Update(func(latest *catalog.Document) error {
		for _, saved := range latest.Repositories {
			if saved.URL == selection.URL && repository.Reference(saved.Reference).Equal(*ref) {
				entry = saved
				redundant = true
				return nil
			}
		}
		latest.Repositories = append(latest.Repositories, entry)
		return nil
	})
	if err != nil {
		context := map[string]any{"url": selection.URL, "reference": ref, "path": path}
		var storage *catalog.StorageError
		if errors.As(err, &storage) && storage.Committed {
			return Result{}, problem.Wrap("configuration saved but failed to finish catalogue update", err, context)
		}
		if cleanupErr := l.Dirs.Remove(path); cleanupErr != nil {
			context["cleanupError"] = cleanupErr.Error()
		}
		return Result{}, problem.Wrap("failed to save configuration after cloning repository", err, context)
	}
	if redundant {
		_ = l.Dirs.Remove(path)
	}
	return addedResult(entry), nil
}

func addedResult(entry catalog.Entry) Result {
	return Result{Status: "added", URL: entry.URL, Reference: repository.Reference(entry.Reference), Path: entry.Path}
}

func gitContext(url, path string, err error) map[string]any {
	context := map[string]any{"url": url, "gitError": err.Error()}
	if path != "" {
		context["path"] = path
	}
	var failure *GitFailure
	if asGitFailure(err, &failure) {
		context["gitCommand"] = fmt.Sprintf("git %s", failure.Command)
	}
	return context
}
