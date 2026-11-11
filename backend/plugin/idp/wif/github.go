package wif

import (
	"strconv"
	"strings"
	"unicode"
)

// githubClaims are the GitHub Actions token claims that name the repository a
// job runs in. Unlike "sub", their format does not change when GitHub moves a
// repository to ID-based subjects ("repo:owner@123/name@456:..."), which is the
// default for repositories created, renamed or transferred after 2026-07-15.
type githubClaims struct {
	RepositoryOwner   string
	RepositoryOwnerID string
	Repository        string
	RepositoryID      string
}

// githubClaimsFrom reads the claims from a token's decoded payload. It is
// lenient on purpose: a claim that is missing or of an unexpected type reads as
// empty and only makes the claims match fail, never the token decode, so tokens
// from other issuers are not affected.
func githubClaimsFrom(payload map[string]any) githubClaims {
	return githubClaims{
		RepositoryOwner:   claimString(payload, "repository_owner"),
		RepositoryOwnerID: claimString(payload, "repository_owner_id"),
		Repository:        claimString(payload, "repository"),
		RepositoryID:      claimString(payload, "repository_id"),
	}
}

// claimString returns a string claim, or a numeric one in decimal, since
// GitHub sends IDs as strings but other issuers may send numbers.
func claimString(payload map[string]any, key string) string {
	switch v := payload[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

// githubSubjectPattern is a subject pattern in GitHub's vocabulary, split into
// the parts the claims carry.
//
//	repo:OWNER[@OWNER_ID]/*
//	repo:OWNER[@OWNER_ID]/REPO[@REPO_ID]:REST
type githubSubjectPattern struct {
	owner   string
	ownerID string
	// repo is empty when the pattern admits every repository of the owner.
	repo   string
	repoID string
	// rest is everything after the repository segment, starting with ":",
	// such as ":*" or ":ref:refs/heads/main". Empty when repo is empty.
	rest string
}

// parseGitHubSubjectPattern reads a GitHub subject pattern. It reports false
// for anything it cannot split cleanly, such as "repo:owner/name*", and those
// patterns are matched by prefix only, as before.
func parseGitHubSubjectPattern(pattern string) (githubSubjectPattern, bool) {
	body, ok := strings.CutPrefix(pattern, githubSubjectPrefix)
	if !ok {
		return githubSubjectPattern{}, false
	}
	ownerPart, remainder, ok := strings.Cut(body, "/")
	if !ok {
		return githubSubjectPattern{}, false
	}
	owner, ownerID, ok := splitNameAndID(ownerPart)
	if !ok {
		return githubSubjectPattern{}, false
	}
	if remainder == "*" {
		return githubSubjectPattern{owner: owner, ownerID: ownerID}, true
	}
	repoPart, rest, ok := strings.Cut(remainder, ":")
	if !ok {
		return githubSubjectPattern{}, false
	}
	repo, repoID, ok := splitNameAndID(repoPart)
	if !ok {
		return githubSubjectPattern{}, false
	}
	return githubSubjectPattern{
		owner:   owner,
		ownerID: ownerID,
		repo:    repo,
		repoID:  repoID,
		rest:    ":" + rest,
	}, true
}

// GitHubRepository is the one repository a GitHub subject pattern names, when
// the pattern admits every job in it.
type GitHubRepository struct {
	Owner   string
	OwnerID string
	Name    string
	ID      string
}

// ParseGitHubRepositoryPattern reads "repo:OWNER[@OWNER_ID]/REPO[@REPO_ID]:*"
// with the grammar the token match uses. It reports false for every other
// pattern, one that names every repository of an owner included. No GitHub name
// holds white space, so a pattern with any matches nothing.
func ParseGitHubRepositoryPattern(pattern string) (GitHubRepository, bool) {
	if strings.ContainsFunc(pattern, unicode.IsSpace) {
		return GitHubRepository{}, false
	}
	p, ok := parseGitHubSubjectPattern(pattern)
	if !ok || p.repo == "" || p.rest != ":*" {
		return GitHubRepository{}, false
	}
	return GitHubRepository{Owner: p.owner, OwnerID: p.ownerID, Name: p.repo, ID: p.repoID}, true
}

// ParseGitLabProjectPattern reads "project_path:GROUP/PROJECT:*", where PROJECT
// may name subgroups, and returns the project's path. It reports false for every
// other pattern. A GitLab pattern is matched by prefix alone, and ":" ends the
// path in GitLab's default subject, project_path:{group}/{project}:ref_type:...
// (docs.gitlab.com/ci/secrets/id_token_authentication), so the pattern admits
// that project and no other. No GitLab path holds "*", ":", "@", white space or
// an empty segment, so a pattern with any of them matches nothing.
func ParseGitLabProjectPattern(pattern string) (string, bool) {
	body, ok := strings.CutPrefix(pattern, gitlabSubjectPrefix)
	if !ok {
		return "", false
	}
	path, ok := strings.CutSuffix(body, ":*")
	if !ok {
		return "", false
	}
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return "", false
	}
	for _, segment := range segments {
		if segment == "" || strings.ContainsAny(segment, "*:@") || strings.ContainsFunc(segment, unicode.IsSpace) {
			return "", false
		}
	}
	return path, true
}

// splitNameAndID splits "name" or "name@123". The name must be non-empty and
// free of wildcards, and the ID, when present, must be all digits.
func splitNameAndID(s string) (string, string, bool) {
	name, id, hasID := strings.Cut(s, "@")
	if name == "" || strings.ContainsAny(name, "*:/") {
		return "", "", false
	}
	if hasID && !isDigits(id) {
		return "", "", false
	}
	return name, id, true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// githubSubjectRest returns what follows the repository segment of a GitHub
// subject, in either format: ":pull_request" for both
// "repo:owner/name:pull_request" and "repo:owner@1/name@2:pull_request".
func githubSubjectRest(subject string) (string, bool) {
	body, ok := strings.CutPrefix(subject, githubSubjectPrefix)
	if !ok {
		return "", false
	}
	_, remainder, ok := strings.Cut(body, "/")
	if !ok {
		return "", false
	}
	i := strings.Index(remainder, ":")
	if i < 0 {
		return "", false
	}
	return remainder[i:], true
}

// matchGitHubClaims is the second way a GitHub token can match a subject
// pattern: the owner, owner ID and repository named in the pattern are compared
// with the token's claims instead of with the "sub" string, so the same pattern
// admits both GitHub subject formats. It never admits a token for another owner
// or repository than the pattern names.
func matchGitHubClaims(subject, pattern string, claims githubClaims) bool {
	if ValidateSubjectPattern(pattern) != nil {
		return false
	}
	p, ok := parseGitHubSubjectPattern(pattern)
	if !ok {
		return false
	}
	if claims.RepositoryOwner == "" || claims.RepositoryOwner != p.owner {
		return false
	}
	if p.ownerID != "" && claims.RepositoryOwnerID != p.ownerID {
		return false
	}
	if p.repo == "" {
		return true
	}
	if claims.Repository != p.owner+"/"+p.repo {
		return false
	}
	if p.repoID != "" && claims.RepositoryID != p.repoID {
		return false
	}
	rest, ok := githubSubjectRest(subject)
	if !ok {
		return false
	}
	if prefix, ok := strings.CutSuffix(p.rest, "*"); ok {
		return strings.HasPrefix(rest, prefix)
	}
	return rest == p.rest
}
