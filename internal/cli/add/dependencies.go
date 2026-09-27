package add

import (
	"context"

	"github.com/isthatcentered/robert/internal/catalog"
)

type Git interface {
	DefaultBranch(context.Context, catalog.RepositoryURL) (string, error)
	Install(context.Context, catalog.RepositoryURL, catalog.Reference, string) error
}

type CheckoutDirs interface {
	Create(string) (string, error)
	Remove(string) error
}
