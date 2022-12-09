package taskcheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
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
func (e *LGTMExecutor) Run(ctx context.Context, _ *api.TaskCheckRun, task *api.Task) (result []api.TaskCheckResult, err error) {
	issue, err := e.store.GetIssueByPipelineID(ctx, task.PipelineID)
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

	if issue.Project.LGTMCheckSetting.Value == api.LGTMValueDisabled {
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
		TypePrefix:  &activityType,
		ContainerID: &issue.ID,
	})
	if err != nil {
		return nil, common.Wrap(err, common.Internal)
	}

	ok, err := checkLGTMcomments(ctx, e.store, activityList, issue)
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

func checkLGTMcomments(ctx context.Context, store *store.Store, activityList []*api.Activity, issue *api.Issue) (bool, error) {
	for _, activity := range activityList {
		ok, err := isCommentLGTM(ctx, store, activity, issue)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func isCommentLGTM(ctx context.Context, store *store.Store, activity *api.Activity, issue *api.Issue) (bool, error) {
	if activity.Comment != "LGTM" {
		return false, nil
	}
	member, err := store.GetProjectMember(ctx, &api.ProjectMemberFind{
		PrincipalID: &activity.CreatorID,
		ProjectID:   &issue.ProjectID,
	})
	if err != nil {
		return false, err
	}
	if member == nil {
		return false, nil
	}
	switch issue.Project.LGTMCheckSetting.Value {
	case api.LGTMValueProjectMember:
		return true, nil
	case api.LGTMValueProjectOwner:
		return member.Role == string(api.Owner), nil
	}
	return false, errors.Errorf("unexpected LGTM setting value: %s", issue.Project.LGTMCheckSetting.Value)
}
