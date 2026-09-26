package add

import (
	"context"

	"github.com/isthatcentered/robert/internal/cli/config"
	"github.com/isthatcentered/robert/internal/cli/repository"
)

type ConfigStore interface {
	Load() (config.Document, bool, error)
	Save(config.Document) error
}

type Git interface {
	DefaultBranch(context.Context, string) (string, error)
	Install(context.Context, string, repository.Reference, string) error
}

type CheckoutDirs interface {
	Create(string) (string, error)
	Remove(string) error
}
