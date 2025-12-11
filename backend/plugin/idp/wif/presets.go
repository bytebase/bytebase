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
func GetPlatformPreset(providerType storepb.ProviderType) *PlatformPreset {
	switch providerType {
	case storepb.ProviderType_PROVIDER_GITHUB:
		return &PlatformPreset{
			IssuerURL:       "https://token.actions.githubusercontent.com",
			AudiencePattern: "https://github.com/%s",
			SubjectPattern:  "repo:%s/%s:ref:refs/heads/%s",
		}
	case storepb.ProviderType_PROVIDER_GITLAB:
		return &PlatformPreset{
			IssuerURL:       "https://gitlab.com",
			AudiencePattern: "https://gitlab.com",
			SubjectPattern:  "project_path:%s/%s:ref_type:branch:ref:%s",
		}
	case storepb.ProviderType_PROVIDER_BITBUCKET:
		return &PlatformPreset{
			IssuerURL:       "https://api.bitbucket.org/2.0/workspaces/%s/pipelines-config/identity/oidc",
			AudiencePattern: "ari:cloud:bitbucket::workspace/%s",
			SubjectPattern:  "%s:*",
		}
	case storepb.ProviderType_PROVIDER_AZURE_DEVOPS:
		return &PlatformPreset{
			IssuerURL:       "https://vstoken.dev.azure.com/%s",
			AudiencePattern: "api://AzureADTokenExchange",
			SubjectPattern:  "sc://%s/%s/%s",
		}
	default:
		return nil
	}
}

// BuildSubjectPattern builds a subject pattern from provider-specific inputs.
func BuildSubjectPattern(providerType storepb.ProviderType, owner, repo, branch string) string {
	preset := GetPlatformPreset(providerType)
	if preset == nil {
		return ""
	}

	switch providerType {
	case storepb.ProviderType_PROVIDER_GITHUB:
		if repo == "" {
			return fmt.Sprintf("repo:%s/*", owner)
		}
		if branch == "" {
			return fmt.Sprintf("repo:%s/%s:*", owner, repo)
		}
		return fmt.Sprintf("repo:%s/%s:ref:refs/heads/%s", owner, repo, branch)
	case storepb.ProviderType_PROVIDER_GITLAB:
		if repo == "" {
			return fmt.Sprintf("project_path:%s/*", owner)
		}
		if branch == "" {
			return fmt.Sprintf("project_path:%s/%s:*", owner, repo)
		}
		return fmt.Sprintf("project_path:%s/%s:ref_type:branch:ref:%s", owner, repo, branch)
	default:
		return ""
	}
}
