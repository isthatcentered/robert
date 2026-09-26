package add

import (
	"context"
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
	Status    string               `json:"status"`
	URL       string               `json:"url"`
	Reference repository.Reference `json:"reference"`
	Path      string               `json:"path"`
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
			return Result{Status: "already_added", URL: entry.URL, Reference: repository.Reference(entry.Reference), Path: entry.Path}, nil
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
	doc.Repositories = append(doc.Repositories, entry)
	if err := l.Catalog.Write(doc); err != nil {
		context := map[string]any{"url": selection.URL, "reference": ref, "path": path}
		if cleanupErr := l.Dirs.Remove(path); cleanupErr != nil {
			context["cleanupError"] = cleanupErr.Error()
		}
		return Result{}, problem.Wrap("failed to save configuration after cloning repository", err, context)
	}
	return Result{Status: "added", URL: entry.URL, Reference: repository.Reference(entry.Reference), Path: entry.Path}, nil
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
