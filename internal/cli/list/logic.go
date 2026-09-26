package list

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Logic struct {
	Config     ConfigStore
	ConfigPath string
}

type Result struct {
	URL       string               `json:"url"`
	Reference repository.Reference `json:"reference"`
	Path      string               `json:"path"`
}

func (l Logic) List(selection Selection) ([]Result, error) {
	doc, _, err := l.Config.Load()
	if err != nil {
		return nil, problem.New("failed to read configuration", map[string]any{
			"path": l.ConfigPath, "cause": err.Error(),
			"hint": "check file permissions and correct the reported configuration fields before retrying",
		})
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
		if selection.Reference != nil && !entry.Reference.Equal(*selection.Reference) {
			continue
		}
		// Load has already validated every timestamp, including unmatched entries.
		addedAt, _ := time.Parse(time.RFC3339Nano, entry.AddedAt)
		matches = append(matches, match{Result{entry.URL, entry.Reference, entry.Path}, addedAt})
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
