package remove

import "github.com/isthatcentered/robert/internal/cli/config"

type ConfigStore interface {
	Load() (config.Document, bool, error)
	Save(config.Document) error
}

type CheckoutDirs interface {
	Remove(string) error
}
