package api

import (
	"fmt"

	"github.com/gosimple/slug"
)

func IssueSlug(issue *Issue) string {
	return fmt.Sprintf("%s-%d", slug.Make(issue.Name), issue.ID)
}

func ProjectSlug(project *Project) string {
	return fmt.Sprintf("%s-%d", slug.Make(project.Name), project.ID)
}

func ProjectWebhookSlug(projectWebhook *ProjectWebhook) string {
	return fmt.Sprintf("%s-%d", slug.Make(projectWebhook.Name), projectWebhook.ID)
}
