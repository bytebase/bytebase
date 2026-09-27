package wif

import (
	"strings"

	"github.com/pkg/errors"
)

// The subject vocabularies this package can reason about. GitHub writes
// "repo:<owner>/<repo>:...", GitLab writes "project_path:<group>/<project>:...".
const (
	githubSubjectPrefix = "repo:"
	gitlabSubjectPrefix = "project_path:"
)

// ValidateSubjectPattern refuses the patterns that match every subject the
// issuer signs. The write paths call it so such a pattern cannot be stored,
// and matchSubjectPattern calls it so one stored before this rule existed
// still matches nothing.
//
// A trailing "*" is a prefix test, so a pattern addressing either vocabulary
// must pin at least the owner segment: "repo:*", "repo:acme*" and "r*" all
// admit every repository on GitHub.
//
// The rule reads the pattern only. provider_type is client-set and says
// nothing about which subjects the issuer signs, so honoring it would let
// "repo*" through on a row that merely claims not to be GitHub.
func ValidateSubjectPattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return errors.New("subject_pattern is required")
	}
	if !strings.HasSuffix(pattern, "*") {
		return nil
	}
	prefix := strings.TrimSuffix(pattern, "*")
	if prefix == "" {
		return errors.New(`subject pattern "*" matches every subject from this issuer`)
	}
	// A prefix carrying "/" has named an owner or a group, which is what the
	// rule asks for.
	if strings.Contains(prefix, "/") {
		return nil
	}
	for _, marker := range []string{githubSubjectPrefix, gitlabSubjectPrefix} {
		// The pattern either spells a marker out, or stops part-way inside one
		// ("r*", "project*") and so still addresses everything written with it.
		if strings.HasPrefix(pattern, marker) || strings.HasPrefix(marker, prefix) {
			return errors.Errorf(
				"subject pattern %q matches every subject from this issuer; name an owner, as in \"repo:my-org/*\" or \"project_path:my-group/*\"",
				pattern)
		}
	}
	// issuer_url is free-form, so a wildcard outside both vocabularies is the
	// operator's call: "system:serviceaccount:prod:*" on EKS names one
	// namespace and carries no "/".
	return nil
}
