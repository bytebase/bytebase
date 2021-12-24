package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) composeRepositoryRelationship(ctx context.Context, repository *api.Repository) error {
	var err error

	repository.Creator, err = s.composePrincipalByID(ctx, repository.CreatorID)
	if err != nil {
		return err
	}

	repository.Updater, err = s.composePrincipalByID(ctx, repository.UpdaterID)
	if err != nil {
		return err
	}

	repository.VCS, err = s.composeVCSByID(ctx, repository.VCSID)
	if err != nil {
		return err
	}

	repository.Project, err = s.composeProjectByID(ctx, repository.ProjectID)
	if err != nil {
		return err
	}

	return nil
}
