package add

import (
	"context"

	"github.com/isthatcentered/robert/internal/catalog"
)

type Git interface {
	DefaultBranch(context.Context, string) (string, error)
	Install(context.Context, string, catalog.Reference, string) error
}

type CheckoutDirs interface {
	Create(string) (string, error)
	Remove(string) error
}
