package add

import (
	"context"

	"github.com/isthatcentered/robert/internal/cli/repository"
)

type Git interface {
	DefaultBranch(context.Context, string) (string, error)
	Install(context.Context, string, repository.Reference, string) error
}

type CheckoutDirs interface {
	Create(string) (string, error)
	Remove(string) error
}
