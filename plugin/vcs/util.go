package vcs

import (
	"fmt"
	"strings"
)

// For now, this method only supports branch reference.
// https://git-scm.com/book/en/v2/Git-Internals-Git-References
func Branch(ref string) (string, error) {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/"), nil
	}

	return "", fmt.Errorf("invalid Git ref: %s", ref)
}
