package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) ComposeRepositoryRelationship(ctx context.Context, repository *api.Repository, includeList []string) error {
	var err error

	repository.Creator, err = s.ComposePrincipalById(context.Background(), repository.CreatorId, includeList)
	if err != nil {
		return err
	}

	repository.Updater, err = s.ComposePrincipalById(context.Background(), repository.UpdaterId, includeList)
	if err != nil {
		return err
	}

	repository.VCS, err = s.ComposeVCSById(context.Background(), repository.VCSId, includeList)
	if err != nil {
		return err
	}

	repository.Project, err = s.ComposeProjectlById(context.Background(), repository.ProjectId, includeList)
	if err != nil {
		return err
	}

	return nil
}
