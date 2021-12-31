package server

import (
	"context"
	"fmt"

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
	// We should always expect VCS to exist when ID isn't the default zero.
	if repository.VCSID > 0 && repository.VCS == nil {
		return fmt.Errorf("VCS not found for ID: %v", repository.VCSID)
	}

	repository.Project, err = s.composeProjectByID(ctx, repository.ProjectID)
	if err != nil {
		return err
	}

	return nil
}
