// Package catalog defines the catalogue domain types and persistence boundary.
package catalog

type Catalog interface {
	// Read returns a snapshot, or defaults without creating a catalogue when absent.
	Read() (Document, error)
	// Update applies change to the latest document and saves it atomically.
	// A callback error aborts without saving. The callback must not call this
	// catalogue again, and must not retain the document after returning.
	Update(change func(*Document) error) error
}

type Document struct {
	Version      int
	InstallDir   string
	Repositories []Entry
}

type Entry struct {
	URL       RepositoryURL
	Path      string
	Reference Reference
	AddedAt   string
}
