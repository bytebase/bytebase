package server

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// map from project ID to the map of <role, principal ID List>.
var testProjectMemberMap = map[int]map[common.ProjectRole][]int{
	100: {
		common.ProjectOwner:     {200},
		common.ProjectDeveloper: {201},
	},
	101: {
		common.ProjectOwner: {202},
	},
}

// map from sheet ID to the project ID.
var testSheetProjectMap = map[int]int{
	1000: 100,
	1001: 100,
	1002: 100,
}

// map from sheet ID to the map of <string, principal ID>
// string can be one of the project roles or "CREATOR" which represents the sheet creator.
var testSheetMemberMap = map[int]map[string]int{
	1000: {
		"OWNER":     200,
		"DEVELOPER": 201,
		"CREATOR":   202,
	},
	1001: {
		"OWNER":     200,
		"DEVELOPER": 201,
		"CREATOR":   202,
	},
	1002: {
		"OWNER":     200,
		"DEVELOPER": 201,
		"CREATOR":   202,
	},
}

// Get any principal with a certain role from the project.
func testFindPrincipalIDFromProject(projectID int, role common.ProjectRole) int {
	m, ok := testProjectMemberMap[projectID]
	if !ok {
		return 999
	}

	ids, ok := m[role]
	if !ok {
		return 999
	}
	if len(ids) == 0 {
		return 999
	}
	return ids[0]
}

func testFindPrincipalIDFromSheet(sheetID int, v string) int {
	m, ok := testSheetMemberMap[sheetID]
	if !ok {
		return 999
	}

	id, ok := m[v]
	if !ok {
		return 999
	}
	return id
}

func getProjectRolesFinderForTest(projectMemberMap map[int]map[common.ProjectRole][]int) func(projectID int, principalID int) (map[common.ProjectRole]bool, error) {
	return func(projectID int, principalID int) (map[common.ProjectRole]bool, error) {
		m, ok := projectMemberMap[projectID]
		if !ok {
			return nil, errors.Errorf("failed to get project iam policy for project %d", projectID)
		}

		projectRoles := make(map[common.ProjectRole]bool)
		for role, ids := range m {
			for _, id := range ids {
				if id == principalID {
					projectRoles[role] = true
				}
			}
		}
		return projectRoles, nil
	}
}

var projectRolesFinderForTest = func(projectID int, principalID int) (map[common.ProjectRole]bool, error) {
	m, ok := testProjectMemberMap[projectID]
	if !ok {
		return nil, errors.Errorf("failed to get project iam policy for project %d", projectID)
	}

	projectRoles := make(map[common.ProjectRole]bool)
	for role, ids := range m {
		for _, id := range ids {
			if id == principalID {
				projectRoles[role] = true
			}
		}
	}
	return projectRoles, nil
}

var sheetFinderForTest = func(sheetID int) (*api.Sheet, error) {
	switch sheetID {
	case 1000:
		return &api.Sheet{
			ID:         1000,
			CreatorID:  testFindPrincipalIDFromSheet(1000, "CREATOR"),
			ProjectID:  testSheetProjectMap[1000],
			Visibility: api.PrivateSheet,
		}, nil
	case 1001:
		return &api.Sheet{
			ID:         1001,
			CreatorID:  testFindPrincipalIDFromSheet(1001, "CREATOR"),
			ProjectID:  testSheetProjectMap[1001],
			Visibility: api.ProjectSheet,
		}, nil
	case 1002:
		return &api.Sheet{
			ID:         1002,
			CreatorID:  testFindPrincipalIDFromSheet(1002, "CREATOR"),
			ProjectID:  testSheetProjectMap[1002],
			Visibility: api.PublicSheet,
		}, nil
	}
	return nil, nil
}
