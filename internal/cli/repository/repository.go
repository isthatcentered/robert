package repository

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

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
