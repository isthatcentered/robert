package catalog

import (
	"fmt"
	"regexp"
	"strings"
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
