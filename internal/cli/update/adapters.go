package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GitCLI struct{}

type gitFailure struct {
	command string
	detail  string
	cause   error
}

func (e *gitFailure) Error() string {
	if e.detail != "" {
		return fmt.Sprintf("git %s: %s", e.command, e.detail)
	}
	return fmt.Sprintf("git %s: %s", e.command, e.cause)
}

func (e *gitFailure) Unwrap() error { return e.cause }

func (GitCLI) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.Join(strings.Fields(string(output)), " ")
		return "", &gitFailure{command: strings.Join(args, " "), detail: detail, cause: err}
	}
	return strings.TrimSpace(string(output)), nil
}

func (g GitCLI) CurrentBranch(ctx context.Context, dir string) (string, error) {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("checkout directory does not exist")
	}
	if err != nil {
		return "", fmt.Errorf("cannot inspect checkout directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("checkout path is not a directory")
	}
	branch, err := g.run(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return branch, nil
}

func (g GitCLI) Head(ctx context.Context, dir string) (string, error) {
	return g.run(ctx, dir, "rev-parse", "--verify", "HEAD")
}

func (g GitCLI) Pull(ctx context.Context, dir string) error {
	_, err := g.run(ctx, dir, "pull", "--ff-only")
	return err
}
