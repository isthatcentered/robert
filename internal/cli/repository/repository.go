package repository

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Reference struct {
	Type   string
	Value  string
	fields map[string]json.RawMessage
}

func (r *Reference) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("reference must be an object")
	}
	var known struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*r = Reference{Type: known.Type, Value: known.Value, fields: fields}
	return nil
}

func (r Reference) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage, len(r.fields)+2)
	for key, value := range r.fields {
		fields[key] = value
	}
	kind, err := json.Marshal(r.Type)
	if err != nil {
		return nil, err
	}
	value, err := json.Marshal(r.Value)
	if err != nil {
		return nil, err
	}
	fields["type"] = kind
	fields["value"] = value
	return json.Marshal(fields)
}

func (r Reference) Equal(other Reference) bool { return r.Type == other.Type && r.Value == other.Value }

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
	if scpRemote.MatchString(input) {
		return input, nil
	}
	parsed, err := url.Parse(input)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https" || parsed.Scheme == "ssh" || parsed.Scheme == "git") && parsed.Host != "" && parsed.Path != "" && parsed.Path != "/" {
		return input, nil
	}
	return "", fmt.Errorf("invalid repository %q: expected owner/repo or an HTTP(S), ssh, git, or scp-style SSH remote URL", input)
}
