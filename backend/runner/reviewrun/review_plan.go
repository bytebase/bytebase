package reviewrun

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

// loadReviewPlan loads the plan and the project behind the issue a review run
// is claimed for.
func loadReviewPlan(ctx context.Context, s *store.Store, projectID string, issueUID int64) (*store.PlanMessage, *store.ProjectMessage, error) {
	issue, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectID}, UID: &issueUID})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get issue")
	}
	if issue == nil {
		return nil, nil, errors.Errorf("issue %d not found in project %s", issueUID, projectID)
	}
	if issue.PlanUID == nil {
		return nil, nil, errors.Errorf("issue %d has no plan", issueUID)
	}
	plan, err := s.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: issue.PlanUID})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return nil, nil, errors.Errorf("plan %d not found in project %s", *issue.PlanUID, projectID)
	}
	project, err := s.GetProjectByResourceID(ctx, projectID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return nil, nil, errors.Errorf("project %s not found", projectID)
	}
	return plan, project, nil
}
