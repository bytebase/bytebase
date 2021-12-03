package api

import (
	"fmt"

	"github.com/gosimple/slug"
)

// IssueSlug is the slug formatter for issuees.
func IssueSlug(issue *Issue) string {
	return fmt.Sprintf("%s-%d", slug.Make(issue.Name), issue.ID)
}

// ProjectSlug is the slug formatter for projects.
func ProjectSlug(project *Project) string {
	return fmt.Sprintf("%s-%d", slug.Make(project.Name), project.ID)
}

// ProjectShortSlug is the slug short formatter for projects.
func ProjectShortSlug(project *Project) string {
	return slug.Make(project.Name)
}

// EnvSlug is the slug formatter for environments.
func EnvSlug(env *Environment) string {
	return slug.Make(env.Name)
}

// ProjectWebhookSlug is the slug formatter for project webhooks.
func ProjectWebhookSlug(projectWebhook *ProjectWebhook) string {
	return fmt.Sprintf("%s-%d", slug.Make(projectWebhook.Name), projectWebhook.ID)
}
