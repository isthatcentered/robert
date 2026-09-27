package catalog

import (
	"strings"
	"testing"
)

func TestReferenceValidation(t *testing.T) {
	for _, kind := range []string{"branch", "tag"} {
		for _, value := range []string{"main", "feature/x", "v1.0", "UPPER", "@", "release/é"} {
			if err := (Reference{kind, value}).Validate(); err != nil {
				t.Errorf("rejected %s %q: %v", kind, value, err)
			}
		}
		for _, value := range []string{"", "bad name", "a\tb", "a\x00b", "a\x7fb", "a..b", "a.lock/b", ".hidden", "a/.hidden", "a@{b", "a//b", "/a", "a/", "a.", "a~b", "a^b", "a:b", "a?b", "a*b", "a[b", "a\\b"} {
			if err := (Reference{kind, value}).Validate(); err == nil {
				t.Errorf("accepted %s %q", kind, value)
			}
		}
	}
	for _, ref := range []Reference{{"branch", "HEAD"}, {"branch", "-main"}, {"commit", "abc"}, {"commit", strings.Repeat("g", 40)}, {"other", "main"}} {
		if err := ref.Validate(); err == nil {
			t.Errorf("accepted reference %+v", ref)
		}
	}
	if err := (Reference{"commit", strings.Repeat("A", 40)}).Validate(); err != nil {
		t.Fatal(err)
	}
}
