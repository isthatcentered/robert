package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/isthatcentered/robert/internal/cli/add"
	"github.com/isthatcentered/robert/internal/cli/config"
	"github.com/isthatcentered/robert/internal/cli/list"
	"github.com/isthatcentered/robert/internal/cli/problem"
	"github.com/isthatcentered/robert/internal/cli/remove"
)

const usage = "robert <add|remove> <repo> [--branch <name> | --tag <name> | --commit <full-40-hex-ID>]\n" + list.Usage

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return writeError(stderr, problem.New("command is required", map[string]any{"usage": usage}))
	}
	if args[0] == "-h" || args[0] == "--help" {
		return writeResult(stdout, stderr, map[string]string{"usage": usage})
	}
	if args[0] != "add" && args[0] != "remove" && args[0] != "list" {
		return writeError(stderr, problem.New("unknown command", map[string]any{"command": args[0], "usage": usage}))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return writeError(stderr, problem.New("failed to find home directory", map[string]any{"cause": err.Error()}))
	}
	path := filepath.Join(home, ".robert")
	store := config.Store{Path: path, Home: home}
	var result any
	switch args[0] {
	case "add":
		logic := add.Logic{Config: store, Git: add.GitCLI{}, Dirs: add.Directory{}, ConfigPath: path}
		result, err = add.Handle(ctx, args[1:], logic)
	case "remove":
		logic := remove.Logic{Config: store, Dirs: remove.Directory{}, ConfigPath: path}
		result, err = remove.Handle(args[1:], logic)
	case "list":
		logic := list.Logic{Config: store, ConfigPath: path}
		result, err = list.Handle(args[1:], logic)
	}
	if err != nil {
		return writeError(stderr, err)
	}
	return writeResult(stdout, stderr, result)
}

func writeResult(stdout, stderr io.Writer, result any) int {
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return writeError(stderr, problem.New("failed to write command result", map[string]any{"cause": err.Error()}))
	}
	return 0
}

func writeError(stderr io.Writer, err error) int {
	_ = json.NewEncoder(stderr).Encode(problem.AsError(err))
	return 1
}
