package tests

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// createVCS creates a VCS.
func (ctl *controller) createVCS(vcsCreate api.VCSCreate) (*api.VCS, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &vcsCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal vcsCreate")
	}

	body, err := ctl.post("/vcs", buf)
	if err != nil {
		return nil, err
	}

	vcs := new(api.VCS)
	if err = jsonapi.UnmarshalPayload(body, vcs); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal vcs response")
	}
	return vcs, nil
}

// createRepository links the repository with the project.
func (ctl *controller) createRepository(repositoryCreate api.RepositoryCreate) (*api.Repository, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &repositoryCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal repositoryCreate")
	}

	body, err := ctl.post(fmt.Sprintf("/project/%d/repository", repositoryCreate.ProjectID), buf)
	if err != nil {
		return nil, err
	}

	repository := new(api.Repository)
	if err = jsonapi.UnmarshalPayload(body, repository); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal repository response")
	}
	return repository, nil
}

// unlinkRepository unlinks the repository from the project by projectID.
func (ctl *controller) unlinkRepository(projectID int) error {
	_, err := ctl.delete(fmt.Sprintf("/project/%d/repository", projectID), nil)
	if err != nil {
		return err
	}
	return nil
}

// listRepository gets the repository list.
func (ctl *controller) listRepository(projectID int) ([]*api.Repository, error) {
	body, err := ctl.get(fmt.Sprintf("/project/%d/repository", projectID), nil)
	if err != nil {
		return nil, err
	}

	var repositories []*api.Repository
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Repository)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get repository response")
	}
	for _, p := range ps {
		repository, ok := p.(*api.Repository)
		if !ok {
			return nil, errors.Errorf("fail to convert repository")
		}
		repositories = append(repositories, repository)
	}
	return repositories, nil
}
