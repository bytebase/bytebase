package server

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

var roleFinder = func(projectID int, principalID int) (common.ProjectRole, error) {
	switch projectID {
	case 100:
		switch principalID {
		case 200:
			return common.ProjectOwner, nil
		case 201:
			return common.ProjectDeveloper, nil
		}
	case 101:
		switch principalID {
		case 200:
			return "", nil
		case 201:
			return "", nil
		case 202:
			return common.ProjectOwner, nil
		}
	}
	return "", nil
}

var sheetFinder = func(sheetID int) (*api.Sheet, error) {
	switch sheetID {
	case 1000:
		return &api.Sheet{
			ID:         1000,
			CreatorID:  202,
			ProjectID:  100,
			Visibility: api.PrivateSheet,
		}, nil
	case 1001:
		return &api.Sheet{
			ID:         1001,
			CreatorID:  202,
			ProjectID:  100,
			Visibility: api.ProjectSheet,
		}, nil
	case 1002:
		return &api.Sheet{
			ID:         1002,
			CreatorID:  202,
			ProjectID:  100,
			Visibility: api.PublicSheet,
		}, nil
	}
	return nil, nil
}
