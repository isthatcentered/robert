package add

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/isthatcentered/robert/internal/catalog"
)

type GitCLI struct{}

type GitFailure struct {
	Command string
	Detail  string
	Cause   error
}

func (e *GitFailure) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Cause.Error()
}

func asGitFailure(err error, target **GitFailure) bool { return errors.As(err, target) }

func (GitCLI) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &GitFailure{Command: strings.Join(args, " "), Detail: strings.TrimSpace(string(output)), Cause: err}
	}
	return string(output), nil
}

func (g GitCLI) DefaultBranch(ctx context.Context, url string) (string, error) {
	output, err := g.run(ctx, "", "ls-remote", "--symref", "--", url, "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "ref: refs/heads/") && strings.HasSuffix(line, "\tHEAD") {
			branch := strings.TrimSuffix(strings.TrimPrefix(line, "ref: refs/heads/"), "\tHEAD")
			if branch != "" {
				return branch, nil
			}
		}
	}
	return "", fmt.Errorf("remote did not advertise a default branch in HEAD")
}

func (g GitCLI) Install(ctx context.Context, url string, ref catalog.Reference, dir string) error {
	steps := [][]string{
		{"init", "--quiet", "--template="},
		{"remote", "add", "origin", url},
	}
	switch ref.Type {
	case "branch":
		local := "refs/remotes/origin/" + ref.Value
		remote := "refs/heads/" + ref.Value
		steps = append(steps,
			[]string{"config", "--replace-all", "remote.origin.fetch", "+" + remote + ":" + local},
			[]string{"fetch", "--depth", "1", "--no-tags", "origin", remote + ":" + local},
			[]string{"checkout", "-b", ref.Value, "--track", local, "--"},
			[]string{"config", "branch." + ref.Value + ".remote", "origin"},
			[]string{"config", "branch." + ref.Value + ".merge", remote},
		)
	case "tag":
		tag := "refs/tags/" + ref.Value
		steps = append(steps,
			[]string{"config", "--replace-all", "remote.origin.fetch", tag + ":" + tag},
			[]string{"fetch", "--depth", "1", "--no-tags", "origin", tag + ":" + tag},
			[]string{"checkout", "--detach", tag + "^{commit}", "--"},
		)
	case "commit":
		steps = append(steps, []string{"fetch", "--depth", "1", "--no-tags", "origin", ref.Value})
	default:
		return fmt.Errorf("unsupported reference type %q", ref.Type)
	}
	for _, args := range steps {
		if _, err := g.run(ctx, dir, args...); err != nil {
			return err
		}
	}
	if ref.Type != "commit" {
		return nil
	}
	objectType, err := g.run(ctx, dir, "cat-file", "-t", "FETCH_HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(objectType) != "commit" {
		return fmt.Errorf("fetched object %s is a %s, not a commit", ref.Value, strings.TrimSpace(objectType))
	}
	fetched, err := g.run(ctx, dir, "rev-parse", "--verify", "FETCH_HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(fetched) != ref.Value {
		return fmt.Errorf("fetched commit %s does not match requested commit %s", strings.TrimSpace(fetched), ref.Value)
	}
	if _, err := g.run(ctx, dir, "checkout", "--detach", "FETCH_HEAD", "--"); err != nil {
		return err
	}
	head, err := g.run(ctx, dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(head) != ref.Value {
		return fmt.Errorf("checked out commit %s does not match requested commit %s", strings.TrimSpace(head), ref.Value)
	}
	return nil
}

type Directory struct{}

func (Directory) Create(root string) (string, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	name := hex.EncodeToString(id[0:4]) + "-" + hex.EncodeToString(id[4:6]) + "-" + hex.EncodeToString(id[6:8]) + "-" + hex.EncodeToString(id[8:10]) + "-" + hex.EncodeToString(id[10:16])
	path := filepath.Join(root, name)
	if err := os.Mkdir(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func (Directory) Remove(path string) error { return os.RemoveAll(path) }
