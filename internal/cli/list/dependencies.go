package list

import "github.com/isthatcentered/robert/internal/cli/config"

type ConfigStore interface {
	// Load validates all saved fields before returning the catalogue.
	Load() (config.Document, bool, error)
}
