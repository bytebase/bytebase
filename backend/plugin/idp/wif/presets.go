package wif

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// PlatformPreset contains default OIDC configuration for a CI/CD platform.
type PlatformPreset struct {
	IssuerURL       string
	AudiencePattern string
	SubjectPattern  string
}

// GetPlatformPreset returns the OIDC preset for a given provider type.
// Currently only GitHub Actions is supported.
func GetPlatformPreset(providerType storepb.ProviderType) *PlatformPreset {
	switch providerType {
	case storepb.ProviderType_PROVIDER_GITHUB:
		return &PlatformPreset{
			IssuerURL:       "https://token.actions.githubusercontent.com",
			AudiencePattern: "https://github.com/%s",
			SubjectPattern:  "repo:%s/%s:ref:refs/heads/%s",
		}
	default:
		return nil
	}
}

// BuildSubjectPattern builds a subject pattern from provider-specific inputs.
// Currently only GitHub Actions is supported.
func BuildSubjectPattern(providerType storepb.ProviderType, owner, repo, branch string) string {
	if providerType != storepb.ProviderType_PROVIDER_GITHUB {
		return ""
	}

	if repo == "" {
		return fmt.Sprintf("repo:%s/*", owner)
	}
	if branch == "" {
		return fmt.Sprintf("repo:%s/%s:*", owner, repo)
	}
	return fmt.Sprintf("repo:%s/%s:ref:refs/heads/%s", owner, repo, branch)
}
