package remove

import (
	"github.com/isthatcentered/robert/internal/cli/arguments"
	"github.com/isthatcentered/robert/internal/cli/problem"
)

const Usage = "robert remove <repo> [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]"

func Handle(args []string, logic Logic) (any, error) {
	selection, err := arguments.Parse(args)
	if err != nil {
		return nil, problem.New(err.Error(), map[string]any{"usage": Usage})
	}
	if selection.Help {
		return map[string]string{"usage": Usage}, nil
	}
	return logic.Remove(selection)
}
