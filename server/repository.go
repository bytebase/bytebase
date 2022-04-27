package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) composeRepositoryRelationship(ctx context.Context, raw *api.RepositoryRaw) (*api.Repository, error) {
	repository := raw.ToRepository()

	creator, err := s.store.GetPrincipalByID(ctx, repository.CreatorID)
	if err != nil {
		return nil, err
	}
	repository.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, repository.UpdaterID)
	if err != nil {
		return nil, err
	}
	repository.Updater = updater

	vcs, err := s.store.GetVCSByID(ctx, repository.VCSID)
	if err != nil {
		return nil, err
	}
	// We should always expect VCS to exist when ID isn't the default zero.
	if repository.VCSID > 0 && vcs == nil {
		return nil, fmt.Errorf("VCS not found for ID: %v", repository.VCSID)
	}
	repository.VCS = vcs

	project, err := s.store.GetProjectByID(ctx, repository.ProjectID)
	if err != nil {
		return nil, err
	}
	repository.Project = project

	return repository, nil
}
