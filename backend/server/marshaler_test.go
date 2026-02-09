package server

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"target", "targets", 1},
		{"target", "target", 0},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestFindSimilar(t *testing.T) {
	candidates := []string{"targets", "title", "description", "changeDatabaseConfig"}

	results := findSimilar("target", candidates, 3)
	if len(results) == 0 {
		t.Fatal("expected at least one suggestion")
	}
	if results[0] != `"targets"` {
		t.Errorf("expected first suggestion to be \"targets\", got %s", results[0])
	}
}
