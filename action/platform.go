package main

import "os"

// JobPlatform represents the supported CI/CD platforms.
type JobPlatform int

const (
	LocalPlatform JobPlatform = iota
	GitHub
	GitLab
	Bitbucket
	AzureDevOps
)

// String returns the string representation of the JobPlatform enum.
func (p JobPlatform) String() string {
	switch p {
	case GitHub:
		return "GitHub"
	case GitLab:
		return "GitLab"
	case Bitbucket:
		return "Bitbucket"
	case AzureDevOps:
		return "Azure DevOps"
	default:
		return "Local"
	}
}

// getJobPlatform returns the platform where the job is running as a JobPlatform.
func getJobPlatform() JobPlatform {
	// https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/store-information-in-variables
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return GitHub
	}
	// https://docs.gitlab.com/ci/variables/predefined_variables/
	if os.Getenv("GITLAB_CI") == "true" {
		return GitLab
	}
	// https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/
	if os.Getenv("BITBUCKET_BUILD_NUMBER") != "" {
		return Bitbucket
	}
	// https://learn.microsoft.com/en-us/azure/devops/pipelines/release/variables?view=azure-devops
	if os.Getenv("SYSTEM_TEAMFOUNDATIONSERVERURI") != "" {
		return AzureDevOps
	}
	return LocalPlatform
}
