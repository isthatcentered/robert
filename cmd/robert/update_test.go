package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateWithLocalGitRemote(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	work := filepath.Join(root, "work")
	bare := filepath.Join(root, "remote.git")
	if err := os.Mkdir(home, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if code, stdout, stderr := updateCommand(); code != 0 || stdout != "No repositories installed. Run \"robert add <repository>\" to install one.\n" || stderr != "" {
		t.Fatalf("empty update = %d, %q, %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".robert")); !os.IsNotExist(err) {
		t.Fatalf("update created catalogue: %v", err)
	}

	git(t, "", "init", "-b", "main", work)
	git(t, work, "config", "user.name", "Test")
	git(t, work, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("initial\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, work, "add", "README.md")
	git(t, work, "commit", "-m", "initial")
	initialID := git(t, work, "rev-parse", "HEAD")
	git(t, work, "tag", "v1")
	git(t, "", "clone", "--bare", work, bare)
	for _, branch := range []string{"other", "missing", "diverge", "ahead", "detached"} {
		git(t, work, "checkout", "-b", branch)
		git(t, work, "push", bare, branch)
	}
	git(t, work, "checkout", "main")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url.file://"+bare+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://example.invalid/demo.git")
	remote := "https://example.invalid/demo.git"

	tagPath := stringField(t, command(t, 0, "add", remote, "--tag", "v1"), "path")
	commitPath := stringField(t, command(t, 0, "add", remote, "--commit", initialID), "path")
	if code, stdout, stderr := updateCommand(); code != 0 || stdout != "No branch-based repositories installed; nothing to update.\n" || stderr != "" {
		t.Fatalf("tag and commit update = %d, %q, %q", code, stdout, stderr)
	}
	mainPath := stringField(t, command(t, 0, "add", remote, "--branch", "main"), "path")
	if code, stdout, stderr := updateCommand(); code != 0 || stdout != "All branch checkouts are already up to date.\n" || stderr != "" {
		t.Fatalf("current branch update = %d, %q, %q", code, stdout, stderr)
	}
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("first update\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, work, "commit", "-am", "first update")
	git(t, work, "push", bare, "main")
	if code, stdout, stderr := updateCommand(); code != 0 || !strings.Contains(stdout, mainPath) || !strings.Contains(stdout, "  updated\n") || stderr != "" {
		t.Fatalf("successful update = %d, %q, %q", code, stdout, stderr)
	}
	if code, stdout, stderr := updateCommand(); code != 0 || stdout != "All branch checkouts are already up to date.\n" || stderr != "" {
		t.Fatalf("repeated update = %d, %q, %q", code, stdout, stderr)
	}
	otherPath := stringField(t, command(t, 0, "add", remote, "--branch", "other"), "path")
	missingPath := stringField(t, command(t, 0, "add", remote, "--branch", "missing"), "path")
	divergePath := stringField(t, command(t, 0, "add", remote, "--branch", "diverge"), "path")
	aheadPath := stringField(t, command(t, 0, "add", remote, "--branch", "ahead"), "path")
	detachedPath := stringField(t, command(t, 0, "add", remote, "--branch", "detached"), "path")

	git(t, otherPath, "checkout", "-b", "feature")
	git(t, detachedPath, "checkout", "--detach")
	if err := os.RemoveAll(missingPath); err != nil {
		t.Fatal(err)
	}
	git(t, divergePath, "config", "user.name", "Test")
	git(t, divergePath, "config", "user.email", "test@example.com")
	git(t, divergePath, "commit", "--allow-empty", "-m", "local")
	git(t, aheadPath, "config", "user.name", "Test")
	git(t, aheadPath, "config", "user.email", "test@example.com")
	git(t, aheadPath, "commit", "--allow-empty", "-m", "ahead")
	git(t, work, "checkout", "diverge")
	git(t, work, "commit", "--allow-empty", "-m", "remote")
	git(t, work, "push", bare, "diverge")
	git(t, work, "checkout", "main")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("updated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, work, "commit", "-am", "updated")
	updatedID := git(t, work, "rev-parse", "HEAD")
	git(t, work, "push", bare, "main")

	configPath := filepath.Join(home, ".robert")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := updateCommand()
	if code != 1 || stderr != "" {
		t.Fatalf("mixed update = %d, %q, %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"REPOSITORY", "REFERENCE", "CHECKOUT", "RESULT",
		otherPath, missingPath, divergePath, detachedPath,
		"Failures:",
		otherPath + ": expected branch other, found feature; switch to other and retry",
		missingPath + ": cannot determine current branch: checkout directory does not exist",
		divergePath + ": cannot update branch diverge: git pull --ff-only:",
		detachedPath + ": expected branch detached, found detached HEAD; switch to detached and retry",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("update output missing %q:\n%s", want, stdout)
		}
	}
	resultTable := strings.SplitN(stdout, "\nFailures:\n", 2)[0]
	rows := strings.Split(strings.TrimSuffix(resultTable, "\n"), "\n")
	if len(rows) != 6 {
		t.Fatalf("result table has %d rows, want header plus five results:\n%s", len(rows), stdout)
	}
	updatedRow := false
	failedRows := 0
	for _, line := range rows {
		if strings.Contains(line, mainPath) && strings.HasSuffix(line, "  updated") {
			updatedRow = true
		}
		for _, path := range []string{otherPath, missingPath, divergePath, detachedPath} {
			if strings.Contains(line, path) && strings.HasSuffix(line, "  failed") {
				failedRows++
			}
		}
	}
	if !updatedRow || failedRows != 4 {
		t.Fatalf("result table missing updated or failed rows:\n%s", stdout)
	}
	if strings.Contains(stdout, tagPath) || strings.Contains(stdout, commitPath) || strings.Contains(stdout, aheadPath) {
		t.Fatalf("ignored or unchanged checkout appeared in results:\n%s", stdout)
	}
	if got := git(t, mainPath, "rev-parse", "HEAD"); got != updatedID {
		t.Fatalf("main HEAD = %s, want %s", got, updatedID)
	}
	if got := git(t, otherPath, "symbolic-ref", "--short", "HEAD"); got != "feature" {
		t.Fatalf("other branch changed to %q", got)
	}
	if got := git(t, divergePath, "log", "-1", "--format=%s"); got != "local" {
		t.Fatalf("diverged branch changed to %q", got)
	}
	after, err := os.ReadFile(configPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("update changed catalogue: %v", err)
	}
}

func TestUpdateHelpAndArguments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if code, stdout, stderr := updateCommand("--help"); code != 0 || !strings.Contains(stdout, "robert update") || stderr != "" {
		t.Fatalf("update help = %d, %q, %q", code, stdout, stderr)
	}
	if code, stdout, stderr := updateCommand("extra"); code != 1 || stdout != "" || !strings.Contains(stderr, "Usage: robert update") {
		t.Fatalf("update argument = %d, %q, %q", code, stdout, stderr)
	}
}

func updateCommand(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), append([]string{"update"}, args...), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
