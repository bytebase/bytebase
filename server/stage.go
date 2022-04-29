package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

// TODO(dragonly): remove this hack
func (s *Server) composeStageRelationshipValidateOnly(ctx context.Context, stage *api.Stage) error {
	var err error
	stage.Creator, err = s.store.GetPrincipalByID(ctx, stage.CreatorID)
	if err != nil {
		return err
	}

	stage.Updater, err = s.store.GetPrincipalByID(ctx, stage.UpdaterID)
	if err != nil {
		return err
	}

	stage.Environment, err = s.store.GetEnvironmentByID(ctx, stage.EnvironmentID)
	if err != nil {
		return err
	}

	for _, task := range stage.TaskList {
		if err := s.composeTaskRelationshipValidateOnly(ctx, task); err != nil {
			return err
		}
	}

	return nil
}
