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
func ProjectSlug(project interface{}) string {
	if p, ok := project.(*Project); ok {
		return fmt.Sprintf("%s-%d", slug.Make(p.Name), p.ID)
	}
	if p, ok := project.(*ProjectPlain); ok {
		return fmt.Sprintf("%s-%d", slug.Make(p.Name), p.ID)
	}
	panic(fmt.Sprintf("invalid project: %v", project))
}

// ProjectShortSlug is the slug short formatter for projects.
func ProjectShortSlug(project interface{}) string {
	if p, ok := project.(*Project); ok {
		return slug.Make(p.Name)
	}
	if p, ok := project.(*ProjectPlain); ok {
		return slug.Make(p.Name)
	}
	panic(fmt.Sprintf("invalid project: %v", project))
}

// EnvSlug is the slug formatter for environments.
func EnvSlug(env *Environment) string {
	return slug.Make(env.Name)
}

// ProjectWebhookSlug is the slug formatter for project webhooks.
func ProjectWebhookSlug(projectWebhook *ProjectWebhook) string {
	return fmt.Sprintf("%s-%d", slug.Make(projectWebhook.Name), projectWebhook.ID)
}
