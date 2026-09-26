package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlagErrorsReturnJSON(t *testing.T) {
	for _, name := range []string{"add", "remove", "list"} {
		t.Run(name, func(t *testing.T) {
			for _, flags := range [][]string{
				{"--unknown"},
				{"--branch"},
				{"--branch", "main", "--tag", "v1"},
				{"--branch", "main", "--branch", "main"},
			} {
				args := []string{name}
				if name != "list" {
					args = append(args, "owner/repo")
				}
				args = append(args, flags...)
				result := command(t, 1, args...)
				if stringField(t, result, "error") == "" {
					t.Fatal("missing error message")
				}
				context, ok := result["context"].(map[string]any)
				if !ok || !strings.HasPrefix(stringField(t, context, "usage"), "robert "+name+" ") {
					t.Fatalf("missing command usage: %v", result)
				}
			}
		})
	}
}

func TestAddAndRemoveWithLocalGitRemote(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	work := filepath.Join(root, "work")
	bare := filepath.Join(root, "remote.git")
	if err := os.Mkdir(home, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, "", "init", "-b", "main", work)
	git(t, work, "config", "user.name", "Test")
	git(t, work, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, work, "add", "README.md")
	git(t, work, "commit", "-m", "main")
	mainID := git(t, work, "rev-parse", "HEAD")
	git(t, work, "tag", "v1.0.0")
	git(t, "", "clone", "--bare", work, bare)
	git(t, work, "checkout", "-b", "develop")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("develop\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, work, "commit", "-am", "develop")
	git(t, work, "push", bare, "develop")

	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url.file://"+bare+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://example.invalid/demo.git")
	remote := "https://example.invalid/demo.git"

	added := command(t, 0, "add", remote)
	assertField(t, added, "status", "added")
	assertReference(t, added, "branch", "main")
	mainPath := stringField(t, added, "path")
	if !filepath.IsAbs(mainPath) {
		t.Fatalf("checkout path is not absolute: %q", mainPath)
	}
	if got := git(t, mainPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); got != "origin/main" {
		t.Fatalf("upstream = %q", got)
	}
	if got := git(t, mainPath, "rev-parse", "--is-shallow-repository"); got != "true" {
		t.Fatalf("branch checkout is not shallow: %q", got)
	}

	configPath := filepath.Join(home, ".robert")
	var doc map[string]any
	readJSON(t, configPath, &doc)
	doc["custom"] = "keep"
	first := doc["repositories"].([]any)[0].(map[string]any)
	first["note"] = "saved field"
	first["reference"].(map[string]any)["source"] = "saved reference field"
	writeJSON(t, configPath, doc)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(mainPath); err != nil {
		t.Fatal(err)
	}
	already := command(t, 0, "add", remote)
	assertField(t, already, "status", "already_added")
	assertField(t, already, "path", mainPath)
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("duplicate add rewrote configuration")
	}

	tag := command(t, 0, "add", remote, "--tag=v1.0.0")
	assertReference(t, tag, "tag", "v1.0.0")
	tagPath := stringField(t, tag, "path")
	if got := git(t, tagPath, "rev-parse", "HEAD"); got != mainID {
		t.Fatalf("tag HEAD = %s, want %s", got, mainID)
	}
	if got := git(t, tagPath, "symbolic-ref", "-q", "HEAD"); got != "" {
		t.Fatalf("tag HEAD is attached: %q", got)
	}

	commit := command(t, 0, "add", remote, "--commit", strings.ToUpper(mainID))
	assertReference(t, commit, "commit", mainID)
	commitPath := stringField(t, commit, "path")
	if got := git(t, commitPath, "rev-parse", "HEAD"); got != mainID {
		t.Fatalf("commit HEAD = %s, want %s", got, mainID)
	}

	git(t, "", "--git-dir", bare, "symbolic-ref", "HEAD", "refs/heads/develop")
	develop := command(t, 0, "add", remote)
	assertReference(t, develop, "branch", "develop")
	if stringField(t, develop, "path") == mainPath {
		t.Fatal("changed default branch reused old installation")
	}

	command(t, 1, "remove", remote)
	removed := command(t, 0, "remove", remote, "--branch", "main")
	assertField(t, removed, "status", "removed")
	assertField(t, removed, "path", mainPath)
	if _, exists := removed["note"]; exists {
		t.Fatal("unknown entry field was retained in removal output")
	}
	if _, exists := removed["reference"].(map[string]any)["source"]; exists {
		t.Fatal("unknown reference field was retained in removal output")
	}
	if stringField(t, removed, "addedAt") == "" {
		t.Fatal("missing saved addedAt")
	}
	doc = nil
	readJSON(t, configPath, &doc)
	if _, exists := doc["custom"]; exists {
		t.Fatal("unknown configuration field was retained")
	}
	if len(doc["repositories"].([]any)) != 3 {
		t.Fatalf("remaining repositories = %v", doc["repositories"])
	}
	if _, err := os.Stat(tagPath); err != nil {
		t.Fatalf("tag checkout removed unexpectedly: %v", err)
	}
}

func TestFirstFailedAddDoesNotCreateConfiguration(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.Mkdir(home, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url.file://"+filepath.Join(root, "missing.git")+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://example.invalid/missing.git")
	response := command(t, 1, "add", "https://example.invalid/missing.git", "--branch", "main")
	assertField(t, response, "error", "failed to clone repository")
	context := response["context"].(map[string]any)
	if context["gitError"] == "" || context["url"] == "" {
		t.Fatalf("missing Git error context: %v", context)
	}
	if _, err := os.Stat(filepath.Join(home, ".robert")); !os.IsNotExist(err) {
		t.Fatalf("configuration created after failed add: %v", err)
	}
	installDir := filepath.Join(home, ".agents", "robert")
	children, err := os.ReadDir(installDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Fatalf("failed checkout remains: %v", children)
	}
}

func TestRemoveRejectsRelativeSavedPathWithoutChangingConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".robert")
	doc := map[string]any{"version": 1, "installDir": filepath.Join(home, "install"), "repositories": []any{map[string]any{"url": "https://example.invalid/demo.git", "path": "relative/path", "reference": map[string]any{"type": "branch", "value": "main"}, "addedAt": "2026-09-26T12:00:00Z"}}}
	writeJSON(t, configPath, doc)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	response := command(t, 1, "remove", "https://example.invalid/demo.git")
	assertField(t, response, "error", "failed to read configuration")
	failureContext := response["context"].(map[string]any)
	if !strings.Contains(stringField(t, failureContext, "cause"), "repositories[0]") || !strings.Contains(stringField(t, failureContext, "cause"), "absolute checkout path") {
		t.Fatalf("missing invalid entry context: %v", failureContext)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("invalid path changed configuration")
	}
}

func command(t *testing.T, expectedCode int, args ...string) map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), args, &stdout, &stderr)
	if code != expectedCode {
		t.Fatalf("robert %v exited %d, want %d; stdout=%s stderr=%s", args, code, expectedCode, stdout.String(), stderr.String())
	}
	data := stdout.Bytes()
	if expectedCode != 0 {
		data = stderr.Bytes()
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON %q: %v", data, err)
	}
	return result
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(args) > 0 && args[0] == "symbolic-ref" && len(output) == 0 {
			return ""
		}
		t.Fatalf("git %v failed: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func assertField(t *testing.T, result map[string]any, key, want string) {
	t.Helper()
	if got := result[key]; got != want {
		t.Fatalf("%s = %v, want %q; result=%v", key, got, want, result)
	}
}

func stringField(t *testing.T, result map[string]any, key string) string {
	t.Helper()
	value, ok := result[key].(string)
	if !ok {
		t.Fatalf("%s is not a string: %v", key, result[key])
	}
	return value
}

func assertReference(t *testing.T, result map[string]any, kind, value string) {
	t.Helper()
	ref, ok := result["reference"].(map[string]any)
	if !ok {
		t.Fatalf("missing reference: %v", result)
	}
	assertField(t, ref, "type", kind)
	assertField(t, ref, "value", value)
}

func readJSON(t *testing.T, path string, result any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, result); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
