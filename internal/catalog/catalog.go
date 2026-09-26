// Package catalog defines the whole-document configuration boundary.
package catalog

type Catalog interface {
	// Read returns defaults without creating a file when no configuration exists.
	Read() (Document, error)
	// Write persists the whole document before returning success.
	Write(Document) error
}

type Document struct {
	Version      int
	InstallDir   string
	Repositories []Entry
}

type Entry struct {
	URL       string
	Path      string
	Reference Reference
	AddedAt   string
}

type Reference struct {
	Type  string
	Value string
}
