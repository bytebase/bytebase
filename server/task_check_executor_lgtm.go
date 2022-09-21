package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

func NewTaskCheckLGTMExecutor() TaskCheckExecutor {
	return &TaskCheckLGTMExecutor{}
}

type TaskCheckLGTMExecutor struct {
}

func (*TaskCheckLGTMExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return []api.TaskCheckResult{}, common.Wrap(err, common.Internal)
	}
	if task == nil {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.BBNamespace,
				Code:      common.Internal.Int(),
				Title:     fmt.Sprintf("Failed to find task %v", taskCheckRun.TaskID),
				Content:   err.Error(),
			},
		}, nil
	}

	activityType := string(api.ActivityIssueCommentCreate)
	activityList, err := server.store.FindActivity(ctx, &api.ActivityFind{
		TypePrefix:  &activityType,
		ContainerID: &task.PipelineID,
	})
	if err != nil {

	}
	for _, activity := range activityList {
		if activity.Comment == "LGTM" {
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusSuccess,
					Namespace: api.BBNamespace,
					Code:      common.Ok.Int(),
					Title:     "OK",
					Content:   "LGTM found!",
				},
			}, nil
		}
	}

	return []api.TaskCheckResult{
		{
			Status:    api.TaskCheckStatusError,
			Namespace: api.BBNamespace,
			Code:      common.NotFound.Int(),
			Title:     "Error",
			Content:   "LGTM NOT FOUND!",
		},
	}, nil
}
