package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) composeRepositoryRelationship(ctx context.Context, raw *api.RepositoryRaw) (*api.Repository, error) {
	var err error

	repository := raw.ToRepository()

	repository.Creator, err = s.composePrincipalByID(ctx, repository.CreatorID)
	if err != nil {
		return nil, err
	}

	repository.Updater, err = s.composePrincipalByID(ctx, repository.UpdaterID)
	if err != nil {
		return nil, err
	}

	repository.VCS, err = s.composeVCSByID(ctx, repository.VCSID)
	if err != nil {
		return nil, err
	}
	// We should always expect VCS to exist when ID isn't the default zero.
	if repository.VCSID > 0 && repository.VCS == nil {
		return nil, fmt.Errorf("VCS not found for ID: %v", repository.VCSID)
	}

	repository.Project, err = s.composeProjectByID(ctx, repository.ProjectID)
	if err != nil {
		return nil, err
	}

	return repository, nil
}
