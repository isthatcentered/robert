package main

import (
	"context"
	"io"
	"os"

	"github.com/isthatcentered/robert/internal/catalog"
	"github.com/isthatcentered/robert/internal/cli/add"
	"github.com/isthatcentered/robert/internal/cli/list"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/remove"
	"github.com/isthatcentered/robert/internal/cli/update"
)

// version is set by GoReleaser through the linker at build time.
var version = "dev"

const help = `Manage your local code reference library for AI agents

Usage:
  robert <command> [arguments]

Commands:
  add       Add a repository to your library
  list      List the repositories in your library
  update    Update the repositories in your library to the latest version
  remove    Remove a repository from your library

Options:
  --version Show the installed version

Examples:
  robert add acme/api
  robert list
  robert update
  robert remove acme/api

Run "robert <command> --help" for command details.`

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return writeError(stderr, problem.New("command is required", nil))
	}
	if args[0] == "-h" || args[0] == "--help" {
		return writeText(stdout, stderr, help+"\n")
	}
	if args[0] == "--version" {
		if len(args) != 1 {
			return writeError(stderr, problem.New("--version does not accept arguments", nil))
		}
		return writeText(stdout, stderr, "robert "+version+"\n")
	}
	if args[0] != "add" && args[0] != "remove" && args[0] != "list" && args[0] != "update" {
		return writeError(stderr, problem.New("unknown command", map[string]any{"command": args[0]}))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return writeError(stderr, problem.New("failed to find home directory", map[string]any{"cause": err.Error()}))
	}
	store := catalog.NewJSONFileCatalog(home)
	var result any
	switch args[0] {
	case "add":
		logic := add.Logic{Catalog: store, Git: add.GitCLI{}, Dirs: add.Directory{}}
		result, err = add.Handle(ctx, args[1:], logic)
	case "remove":
		logic := remove.Logic{Catalog: store, Dirs: remove.Directory{}}
		result, err = remove.Handle(args[1:], logic)
	case "list":
		logic := list.Logic{Catalog: store}
		result, err = list.Handle(args[1:], logic)
	case "update":
		logic := update.Logic{Catalog: store, Git: update.GitCLI{}}
		result, err = update.Handle(ctx, args[1:], logic)
	}
	if err != nil {
		return writeError(stderr, err)
	}
	var output, warning string
	failed := false
	switch value := result.(type) {
	case string:
		output = value + "\n"
	case add.Result:
		output = add.Format(value)
	case *remove.Result:
		output = remove.Format(value)
		warning = remove.FormatWarning(value)
	case list.Output:
		output = list.Format(value)
	case update.Output:
		output = update.Format(value)
		failed = value.Failed()
	default:
		return writeError(stderr, problem.New("failed to format command result", nil))
	}
	if code := writeText(stdout, stderr, output); code != 0 {
		return code
	}
	if warning != "" {
		_, _ = io.WriteString(stderr, warning)
	}
	if failed {
		return 1
	}
	return 0
}

func writeText(stdout, stderr io.Writer, output string) int {
	if _, err := io.WriteString(stdout, output); err != nil {
		return writeError(stderr, problem.New("failed to write command result", map[string]any{"cause": err.Error()}))
	}
	return 0
}

func writeError(stderr io.Writer, err error) int {
	_, _ = io.WriteString(stderr, problem.Format(err))
	return 1
}
