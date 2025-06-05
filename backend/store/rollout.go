package store

import (
	"context"

	"github.com/pkg/errors"
)

// GetRollout gets the rollout by rollout ID.
func (s *Store) GetRollout(ctx context.Context, rolloutID int) (*PipelineMessage, error) {
	tasks, err := s.ListTasks(ctx, &TaskFind{PipelineID: &rolloutID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find tasks for pipeline %d", rolloutID)
	}
	pipeline, err := s.GetPipelineV2ByID(ctx, rolloutID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pipeline")
	}
	if pipeline == nil {
		return nil, nil
	}
	rollout := *pipeline
	rollout.Tasks = tasks

	return &rollout, nil
}
