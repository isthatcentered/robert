package update

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/isthatcentered/robert/internal/catalog"
)

type Logic struct {
	Catalog catalog.Catalog
	Git     Git
}

type Result struct {
	URL       catalog.RepositoryURL
	Reference catalog.Reference
	Path      string
	Status    string
	Failure   string
}

type Output struct {
	Results        []Result
	EmptyCatalogue bool
	NoBranches     bool
}

func (o Output) Failed() bool {
	for _, result := range o.Results {
		if result.Status == "failed" {
			return true
		}
	}
	return false
}

func Format(output Output) string {
	if output.EmptyCatalogue {
		return "No repositories installed. Run \"robert add <repository>\" to install one.\n"
	}
	if output.NoBranches {
		return "No branch-based repositories installed; nothing to update.\n"
	}
	if len(output.Results) == 0 {
		return "All branch checkouts are already up to date.\n"
	}
	const repositoryHeader = "REPOSITORY"
	const referenceHeader = "REFERENCE"
	const checkoutHeader = "CHECKOUT"
	repositoryWidth := len(repositoryHeader)
	referenceWidth := len(referenceHeader)
	checkoutWidth := len(checkoutHeader)
	for _, result := range output.Results {
		repositoryWidth = max(repositoryWidth, utf8.RuneCountInString(string(result.URL)))
		referenceWidth = max(referenceWidth, utf8.RuneCountInString(result.Reference.Type)+1+utf8.RuneCountInString(result.Reference.Value))
		checkoutWidth = max(checkoutWidth, utf8.RuneCountInString(result.Path))
	}
	var text strings.Builder
	fmt.Fprintf(&text, "%-*s  %-*s  %-*s  RESULT\n", repositoryWidth, repositoryHeader, referenceWidth, referenceHeader, checkoutWidth, checkoutHeader)
	for _, result := range output.Results {
		reference := result.Reference.Type + " " + result.Reference.Value
		fmt.Fprintf(&text, "%-*s  %-*s  %-*s  %s\n", repositoryWidth, result.URL, referenceWidth, reference, checkoutWidth, result.Path, result.Status)
	}
	if output.Failed() {
		text.WriteString("\nFailures:\n")
		for _, result := range output.Results {
			if result.Status == "failed" {
				fmt.Fprintf(&text, "  %s: %s\n", result.Path, result.Failure)
			}
		}
	}
	return text.String()
}

func (l Logic) Update(ctx context.Context) (Output, error) {
	doc, err := l.Catalog.Read()
	if err != nil {
		return Output{}, err
	}
	if len(doc.Repositories) == 0 {
		return Output{EmptyCatalogue: true}, nil
	}
	entries := make([]catalog.Entry, 0, len(doc.Repositories))
	for _, entry := range doc.Repositories {
		if entry.Reference.Type == "branch" {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return Output{NoBranches: true}, nil
	}
	slices.SortFunc(entries, compareEntries)
	output := Output{Results: make([]Result, 0, len(entries))}
	for _, entry := range entries {
		result := l.updateOne(ctx, entry)
		if result.Status != "current" {
			output.Results = append(output.Results, result)
		}
	}
	return output, nil
}

func compareEntries(a, b catalog.Entry) int {
	addedA, _ := time.Parse(time.RFC3339Nano, a.AddedAt)
	addedB, _ := time.Parse(time.RFC3339Nano, b.AddedAt)
	return cmp.Or(
		cmp.Compare(a.URL, b.URL),
		addedA.Compare(addedB),
		strings.Compare(a.Reference.Type, b.Reference.Type),
		strings.Compare(a.Reference.Value, b.Reference.Value),
		strings.Compare(a.Path, b.Path),
	)
}

func (l Logic) updateOne(ctx context.Context, entry catalog.Entry) Result {
	result := Result{URL: entry.URL, Reference: entry.Reference, Path: entry.Path}
	failed := func(detail string) Result {
		result.Status = "failed"
		result.Failure = detail
		return result
	}
	branch, err := l.Git.CurrentBranch(ctx, entry.Path)
	if err != nil {
		return failed(fmt.Sprintf("cannot determine current branch: %s", err))
	}
	if branch != entry.Reference.Value {
		found := branch
		if found == "" {
			found = "detached HEAD"
		}
		return failed(fmt.Sprintf("expected branch %s, found %s; switch to %s and retry", entry.Reference.Value, found, entry.Reference.Value))
	}
	before, err := l.Git.Head(ctx, entry.Path)
	if err != nil {
		return failed(fmt.Sprintf("cannot read HEAD before pull: %s", err))
	}
	if err := l.Git.Pull(ctx, entry.Path); err != nil {
		return failed(fmt.Sprintf("cannot update branch %s: %s", entry.Reference.Value, err))
	}
	after, err := l.Git.Head(ctx, entry.Path)
	if err != nil {
		return failed(fmt.Sprintf("cannot read HEAD after pull: %s", err))
	}
	if after == before {
		result.Status = "current"
	} else {
		result.Status = "updated"
	}
	return result
}
