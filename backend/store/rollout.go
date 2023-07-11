package store

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// GetRollout gets the rollout by rollout ID.
func (s *Store) GetRollout(ctx context.Context, rolloutID int) (*PipelineMessage, error) {
	tasks, err := s.ListTasks(ctx, &api.TaskFind{PipelineID: &rolloutID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find tasks for pipeline %d", rolloutID)
	}
	stages, err := s.ListStageV2(ctx, rolloutID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find stage list")
	}
	pipline, err := s.GetPipelineV2ByID(ctx, rolloutID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pipeline")
	}
	rollout := *pipline
	rollout.Stages = stages

	for _, stage := range stages {
		for _, task := range tasks {
			if task.StageID == stage.ID {
				stage.TaskList = append(stage.TaskList, task)
			}
		}
	}

	return &rollout, nil
}
