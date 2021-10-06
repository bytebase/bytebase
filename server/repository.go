package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) ComposeRepositoryRelationship(ctx context.Context, repository *api.Repository) error {
	var err error

	repository.Creator, err = s.ComposePrincipalById(ctx, repository.CreatorId)
	if err != nil {
		return err
	}

	repository.Updater, err = s.ComposePrincipalById(ctx, repository.UpdaterId)
	if err != nil {
		return err
	}

	repository.VCS, err = s.ComposeVCSById(ctx, repository.VCSId)
	if err != nil {
		return err
	}

	repository.Project, err = s.ComposeProjectlById(ctx, repository.ProjectId)
	if err != nil {
		return err
	}

	return nil
}
