// Package vcs provides the utilities for VCS plugins.
package vcs

import (
	"strings"

	"github.com/pkg/errors"
)

// Branch is the helper function returns the branch name from reference name.
// For now, this method only supports branch reference.
// https://git-scm.com/book/en/v2/Git-Internals-Git-References
func Branch(ref string) (string, error) {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/"), nil
	}

	return "", errors.Errorf("invalid Git ref: %s", ref)
}
