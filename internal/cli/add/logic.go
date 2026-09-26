package add

import (
	"context"
	"fmt"
	"time"

	"github.com/isthatcentered/robert/internal/cli/config"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Logic struct {
	Config     ConfigStore
	Git        Git
	Dirs       CheckoutDirs
	ConfigPath string
	Now        func() time.Time
}

type Result struct {
	Status    string               `json:"status"`
	URL       string               `json:"url"`
	Reference repository.Reference `json:"reference"`
	Path      string               `json:"path"`
}

func (l Logic) Add(ctx context.Context, selection Selection) (Result, error) {
	doc, _, err := l.Config.Load()
	if err != nil {
		return Result{}, problem.New("failed to read configuration", map[string]any{"path": l.ConfigPath, "cause": err.Error(), "hint": "check file permissions and correct the reported configuration fields before retrying"})
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
		if entry.URL == selection.URL && entry.Reference.Equal(*ref) {
			return Result{Status: "already_added", URL: entry.URL, Reference: entry.Reference, Path: entry.Path}, nil
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
	entry := config.Entry{URL: selection.URL, Path: path, Reference: *ref, AddedAt: now().UTC().Format(time.RFC3339Nano)}
	doc.Repositories = append(doc.Repositories, entry)
	if err := l.Config.Save(doc); err != nil {
		context := map[string]any{"url": selection.URL, "reference": ref, "path": path, "configPath": l.ConfigPath, "cause": err.Error()}
		if cleanupErr := l.Dirs.Remove(path); cleanupErr != nil {
			context["cleanupError"] = cleanupErr.Error()
		}
		return Result{}, problem.New("failed to save configuration after cloning repository", context)
	}
	return Result{Status: "added", URL: entry.URL, Reference: entry.Reference, Path: entry.Path}, nil
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
