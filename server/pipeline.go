package server

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) schedulePipelineTaskCheck(ctx context.Context, pipeline *api.Pipeline) error {
	var createList []*api.TaskCheckRunCreate
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			create, err := s.TaskCheckScheduler.getTaskCheck(ctx, task, api.SystemBotID)
			if err != nil {
				return errors.Wrapf(err, "failed to get task check for task %d", task.ID)
			}
			createList = append(createList, create...)
		}
	}
	if _, err := s.store.BatchCreateTaskCheckRun(ctx, createList); err != nil {
		return errors.Wrap(err, "failed to batch insert TaskCheckRunCreate")
	}
	return nil
}
