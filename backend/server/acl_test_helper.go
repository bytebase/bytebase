package server

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
