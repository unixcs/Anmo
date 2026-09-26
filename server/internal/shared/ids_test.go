package shared

import (
	"regexp"
	"testing"
)

func TestNewID(t *testing.T) {
	seen := map[string]bool{}
	re := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	for i := 0; i < 1000; i++ {
		id := NewID()
		if len(id) != 26 || !re.MatchString(id) {
			t.Fatalf("bad id %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
