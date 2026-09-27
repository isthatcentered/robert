package list

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Logic struct {
	Catalog catalog.Catalog
}

type Result struct {
	URL       string
	Reference repository.Reference
	Path      string
}

func Format(output Output) string {
	if len(output.Results) == 0 {
		if output.Filtered {
			return "No repositories match.\n"
		}
		return "No repositories found.\n"
	}
	const repositoryHeader = "REPOSITORY"
	const referenceHeader = "REFERENCE"
	const checkoutHeader = "CHECKOUT"
	repositoryWidth := len(repositoryHeader)
	referenceWidth := len(referenceHeader)
	for _, result := range output.Results {
		repositoryWidth = max(repositoryWidth, utf8.RuneCountInString(result.URL))
		referenceWidth = max(referenceWidth, utf8.RuneCountInString(result.Reference.Type)+1+utf8.RuneCountInString(result.Reference.Value))
	}
	var text strings.Builder
	fmt.Fprintf(&text, "%-*s  %-*s  %s\n", repositoryWidth, repositoryHeader, referenceWidth, referenceHeader, checkoutHeader)
	for _, result := range output.Results {
		reference := result.Reference.Type + " " + result.Reference.Value
		fmt.Fprintf(&text, "%-*s  %-*s  %s\n", repositoryWidth, result.URL, referenceWidth, reference, result.Path)
	}
	return text.String()
}

func (l Logic) List(selection Selection) ([]Result, error) {
	doc, err := l.Catalog.Read()
	if err != nil {
		return nil, err
	}
	type match struct {
		result  Result
		addedAt time.Time
	}
	var matches []match
	search := strings.ToLower(strings.TrimSpace(selection.Search))
	for _, entry := range doc.Repositories {
		if !strings.Contains(strings.ToLower(repository.SearchPath(entry.URL)), search) {
			continue
		}
		if selection.Reference != nil && !repository.Reference(entry.Reference).Equal(*selection.Reference) {
			continue
		}
		// Saved configuration is authoritative; timestamps are trusted to be valid.
		addedAt, _ := time.Parse(time.RFC3339Nano, entry.AddedAt)
		matches = append(matches, match{Result{entry.URL, repository.Reference(entry.Reference), entry.Path}, addedAt})
	}
	slices.SortFunc(matches, func(a, b match) int {
		return cmp.Or(
			strings.Compare(a.result.URL, b.result.URL),
			a.addedAt.Compare(b.addedAt),
			strings.Compare(a.result.Reference.Type, b.result.Reference.Type),
			strings.Compare(a.result.Reference.Value, b.result.Reference.Value),
			strings.Compare(a.result.Path, b.result.Path),
		)
	})
	results := make([]Result, 0, len(matches))
	for _, match := range matches {
		results = append(results, match.result)
	}
	return results, nil
}
