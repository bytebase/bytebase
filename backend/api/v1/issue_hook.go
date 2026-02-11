package v1

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// postCreateIssue runs post-creation logic for an issue: webhook, approval finding, and auto-approve.
func postCreateIssue(
	ctx context.Context,
	stores *store.Store,
	webhookManager *webhook.Manager,
	licenseService *enterprise.LicenseService,
	b *bus.Bus,
	project *store.ProjectMessage,
	creatorName string,
	creatorEmail string,
	issue *store.IssueMessage,
) (*store.IssueMessage, error) {
	// Trigger ISSUE_CREATED webhook.
	webhookManager.CreateEvent(ctx, &webhook.Event{
		Type:    storepb.Activity_ISSUE_CREATED,
		Project: webhook.NewProject(project),
		IssueCreated: &webhook.EventIssueCreated{
			Creator: &webhook.User{
				Name:  creatorName,
				Email: creatorEmail,
			},
			Issue: webhook.NewIssue(issue),
		},
	})

	// Trigger approval finding based on issue type.
	switch issue.Type {
	case
		storepb.Issue_ACCESS_GRANT,
		storepb.Issue_GRANT_REQUEST,
		storepb.Issue_DATABASE_EXPORT:

		if err := approval.FindAndApplyApprovalTemplate(ctx, stores, webhookManager, licenseService, issue); err != nil {
			slog.Error("failed to find approval template",
				slog.Int("issue_uid", issue.UID),
				slog.String("issue_title", issue.Title),
				log.BBError(err))
		}

		// Refresh issue to get updated approval payload.
		uid := issue.UID
		var err error
		issue, err = stores.GetIssue(ctx, &store.FindIssueMessage{UID: &uid})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to refresh issue")
		}

		if issue.Type == storepb.Issue_DATABASE_EXPORT {
			return issue, nil
		}

		// For ACCESS_GRANT/GRANT_REQUEST that is auto-approved (no approval template), complete it.
		approved, err := utils.CheckApprovalApproved(issue.Payload.Approval)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check if approval is approved")
		}
		if !approved {
			return issue, nil
		}
		issue, err = completeAccessRequestIssue(ctx, stores, creatorEmail, issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to complete grant request")
		}
	case storepb.Issue_DATABASE_CHANGE:
		b.ApprovalCheckChan <- int64(issue.UID)
	default:
	}

	return issue, nil
}

// completeAccessRequestIssue completes the ACCESS_GRANT/GRANT_REQUEST issue.
// For GRANT_REQUEST issue: grant the privilege and updating the status.
// For ACCESS_GRANT issue: mark the status as ACTIVE.
func completeAccessRequestIssue(ctx context.Context, stores *store.Store, userEmail string, issue *store.IssueMessage) (*store.IssueMessage, error) {
	switch issue.Type {
	case storepb.Issue_ACCESS_GRANT:
		if issue.Payload.AccessGrantId == "" {
			return nil, errors.Errorf("invalid access grant id for issue %d", issue.UID)
		}
		status := storepb.AccessGrant_ACTIVE
		if _, err := stores.UpdateAccessGrant(ctx, issue.Payload.AccessGrantId, &store.UpdateAccessGrantMessage{
			Status: &status,
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to update access grant status")
		}
	case storepb.Issue_GRANT_REQUEST:
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, stores, issue, issue.Payload.GrantRequest); err != nil {
			return nil, err
		}
	default:
		return issue, nil
	}

	newStatus := storepb.Issue_DONE
	updatedIssue, err := stores.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{Status: &newStatus})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	if _, err := stores.CreateIssueComments(ctx, userEmail, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Event: &storepb.IssueCommentPayload_IssueUpdate_{
				IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
					FromStatus: &issue.Status,
					ToStatus:   &updatedIssue.Status,
				},
			},
		},
	}); err != nil {
		slog.Warn("failed to create issue comment after changing the issue status", log.BBError(err))
	}

	return updatedIssue, nil
}
