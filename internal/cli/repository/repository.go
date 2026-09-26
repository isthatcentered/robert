package repository

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

type Reference struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (r Reference) Equal(other Reference) bool {
	if r.Type != other.Type {
		return false
	}
	if r.Type == "commit" {
		return strings.EqualFold(r.Value, other.Value)
	}
	return r.Value == other.Value
}

var commitID = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func (r Reference) Validate() error {
	switch r.Type {
	case "commit":
		if !commitID.MatchString(r.Value) {
			return fmt.Errorf("commit requires a full 40-character hexadecimal ID")
		}
		return nil
	case "branch", "tag":
		if validRefName(r.Value) && (r.Type != "branch" || (r.Value != "HEAD" && !strings.HasPrefix(r.Value, "-"))) {
			return nil
		}
		return fmt.Errorf("%s value %q must be a valid nonempty Git reference name", r.Type, r.Value)
	default:
		return fmt.Errorf("type %q must be branch, tag, or commit", r.Type)
	}
}

func validRefName(value string) bool {
	if value == "" || strings.HasSuffix(value, ".") || strings.Contains(value, "..") || strings.Contains(value, "@{") {
		return false
	}
	for _, char := range value {
		if char <= ' ' || char == 127 || strings.ContainsRune("~^:?*[\\", char) {
			return false
		}
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}

// SearchPath returns the repository namespace and name, without its transport or host.
// Callers pass remote URLs already validated by NormalizeURL.
func SearchPath(remote string) string {
	var path string
	if scpRemote.MatchString(remote) {
		_, path, _ = strings.Cut(remote, ":")
	} else {
		parsed, err := url.Parse(remote)
		if err != nil {
			return ""
		}
		path = parsed.Path
	}
	return strings.TrimSuffix(strings.Trim(path, "/"), ".git")
}

var shorthand = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var scpRemote = regexp.MustCompile(`^[^@/:\s]+@[^@/:\s]+:.+$`)

func NormalizeURL(input string) (string, error) {
	if shorthand.MatchString(input) {
		parts := strings.Split(input, "/")
		if parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
			return "", fmt.Errorf("invalid repository %q: expected owner/repo or a remote URL", input)
		}
		return "https://github.com/" + input + ".git", nil
	}
	if scpRemote.MatchString(input) && validRemotePath(strings.SplitN(input, ":", 2)[1]) {
		return input, nil
	}
	parsed, err := url.Parse(input)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https" || parsed.Scheme == "ssh" || parsed.Scheme == "git") && parsed.Hostname() != "" && validRemotePath(parsed.Path) {
		return input, nil
	}
	return "", fmt.Errorf("invalid repository %q: expected owner/repo or an HTTP(S), ssh, git, or scp-style SSH remote URL", input)
}

func validRemotePath(path string) bool {
	path = strings.Trim(path, "/")
	return path != "" && path != ".git" && path != "." && path != ".." && !strings.ContainsRune(path, 0) && strings.IndexFunc(path, unicode.IsControl) == -1
}
