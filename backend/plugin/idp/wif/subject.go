package wif

import (
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
// A trailing "*" is a prefix test, so a pattern in one of the two vocabularies
// we model must pin at least the owner segment: "repo:*", "repo:acme*" and
// "r*" all admit every repository on GitHub.
func ValidateSubjectPattern(providerType storepb.WorkloadIdentityConfig_ProviderType, pattern string) error {
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
	if inSubjectVocabulary(providerType, pattern, prefix,
		githubSubjectPrefix, storepb.WorkloadIdentityConfig_GITHUB) {
		return errors.Errorf("subject pattern %q matches every repository from this issuer; name an owner, as in \"repo:my-org/*\"", pattern)
	}
	if inSubjectVocabulary(providerType, pattern, prefix,
		gitlabSubjectPrefix, storepb.WorkloadIdentityConfig_GITLAB) {
		return errors.Errorf("subject pattern %q matches every project from this issuer; name a group, as in \"project_path:my-group/*\"", pattern)
	}
	// issuer_url is free-form, so a wildcard outside both vocabularies is the
	// operator's call: "system:serviceaccount:prod:*" on EKS names one
	// namespace and carries no "/".
	return nil
}

// inSubjectVocabulary reports whether a wildcard pattern addresses the subjects
// written with marker. A pattern that spells the marker out says so itself. A
// prefix that stops inside one ("r*", "project*") needs the row to name its
// provider, and a row that names none predates that requirement, so it is
// judged against both markers rather than neither.
func inSubjectVocabulary(
	providerType storepb.WorkloadIdentityConfig_ProviderType,
	pattern, prefix, marker string,
	provider storepb.WorkloadIdentityConfig_ProviderType,
) bool {
	if strings.HasPrefix(pattern, marker) {
		return true
	}
	if !strings.HasPrefix(marker, prefix) {
		return false
	}
	return providerType == provider ||
		providerType == storepb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED
}
