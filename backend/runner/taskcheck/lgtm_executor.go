package taskcheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// NewLGTMExecutor creates a task check LGTM executor.
func NewLGTMExecutor(store *store.Store) Executor {
	return &LGTMExecutor{
		store: store,
	}
}

// LGTMExecutor is the task check LGTM executor. It checks if "LGTM" comments are present.
type LGTMExecutor struct {
	store *store.Store
}

// Run will run the task check LGTM executor once.
func (e *LGTMExecutor) Run(ctx context.Context, _ *store.TaskCheckRunMessage, task *store.TaskMessage) (result []api.TaskCheckResult, err error) {
	issue, err := e.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return nil, common.Wrap(err, common.Internal)
	}
	if issue == nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     fmt.Sprintf("Failed to find issue by pipelineID %d", task.PipelineID),
			},
		}, nil
	}

	project := issue.Project
	if project.LGTMCheckSetting.Value == api.LGTMValueDisabled {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "Skip check",
				Content:   "This check is disabled.",
			},
		}, nil
	}

	activityType := string(api.ActivityIssueCommentCreate)
	activityList, err := e.store.FindActivity(ctx, &api.ActivityFind{
		TypePrefixList: []string{activityType},
		ContainerID:    &issue.UID,
	})
	if err != nil {
		return nil, common.Wrap(err, common.Internal)
	}

	ok, err := checkLGTMcomments(ctx, e.store, activityList, project)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check LGTM comments")
	}
	if ok {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "OK",
				Content:   "Valid LGTM found",
			},
		}, nil
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusError,
			Namespace: api.BBNamespace,
			Code:      common.NotFound.Int(),
			Title:     "Not Found",
			Content:   "Valid LGTM NOT found",
		},
	}, nil
}

func checkLGTMcomments(ctx context.Context, store *store.Store, activityList []*api.Activity, project *store.ProjectMessage) (bool, error) {
	for _, activity := range activityList {
		ok, err := isCommentLGTM(ctx, store, activity, project)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func isCommentLGTM(ctx context.Context, stores *store.Store, activity *api.Activity, project *store.ProjectMessage) (bool, error) {
	if activity.Comment != "LGTM" {
		return false, nil
	}

	policy, err := stores.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &project.UID})
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", project.UID)
	}
	role := api.UnknownRole
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member.ID == activity.CreatorID {
				role = binding.Role
			}
		}
	}

	switch project.LGTMCheckSetting.Value {
	case api.LGTMValueProjectMember:
		return true, nil
	case api.LGTMValueProjectOwner:
		return role == api.Owner, nil
	}
	return false, errors.Errorf("unexpected LGTM setting value: %s", project.LGTMCheckSetting.Value)
}
