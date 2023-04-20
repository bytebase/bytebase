package server

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

func TestEnforceWorkspaceDeveloperDatabaseRouteACL(t *testing.T) {
	type test struct {
		desc        string
		path        string
		body        string
		method      string
		principalID int
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Retrieve the database which is unassigned to any project",
			path:        "/database/301",
			method:      "GET",
			body:        "",
			principalID: 201,
			errMsg:      "user is not a member of project owns this database",
		},
		{
			desc:        "Retrieve the database which is assigned to a project, but the user is not a member of the project",
			path:        "/database/302",
			method:      "GET",
			body:        "",
			principalID: 201,
			errMsg:      "user is not a member of project owns this database",
		},
		{
			desc:        "Retrieve the database which is assigned to a project, and the user is a member of the project",
			path:        "/database/303",
			method:      "GET",
			body:        "",
			principalID: 201,
			errMsg:      "",
		},
		{
			desc:        "Transfer the database to the project which the user is not a member of",
			path:        "/database/303",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":101}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "user is not a member of project 101",
		},
		{
			desc:        "Transfer out the database under the project which the user is not a member of",
			path:        "/database/302",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":102}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "user is not a member of project owns the database 302",
		},
		{
			desc:        "Transfer out the database under the project which the user is not a member of",
			path:        "/database/304",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":103}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperDatabaseRouteACL(tc.path, tc.method, tc.body, tc.principalID, testWorkspaceDeveloperDatabaseRouteMockGetDatabaseProjectID, getProjectRolesFinderForTest(testWorkspaceDeveloperDatabaseRouteHelper.projectMembers))
			if err != nil {
				if tc.errMsg == "" {
					t.Errorf("expect no error, got %s", err.Message)
				} else if tc.errMsg != err.Message {
					t.Errorf("expect error %s, got %s", tc.errMsg, err.Message)
				}
			} else if tc.errMsg != "" {
				t.Errorf("expect error %s, got no error", tc.errMsg)
			}
		})
	}
}

var testWorkspaceDeveloperDatabaseRouteHelper = struct {
	projectMembers map[int]map[common.ProjectRole][]int
	// projectOwnsDatabase is a map from project ID to the map of <database ID, true>.
	projectOwnsDatabase map[int]map[int]bool
}{
	projectMembers: map[int]map[common.ProjectRole][]int{
		1:   {},
		101: {},
		102: {
			common.ProjectDeveloper: {201},
		},
		103: {
			common.ProjectDeveloper: {201},
		},
	},
	projectOwnsDatabase: map[int]map[int]bool{
		// Default Project owns database 301.
		1: {
			301: true,
		},
		// Project 101 owns database 302.
		101: {
			302: true,
		},
		// Project 102 owns database 303.
		102: {
			303: true,
		},
		// Project 103 owns database 304.
		103: {
			304: true,
		},
	},
}

func testWorkspaceDeveloperDatabaseRouteMockGetDatabaseProjectID(databaseID int) (int, error) {
	for projectID, databaseMap := range testWorkspaceDeveloperDatabaseRouteHelper.projectOwnsDatabase {
		_, ok := databaseMap[databaseID]
		if ok {
			return projectID, nil
		}
	}
	return 0, errors.Errorf("database %d not found", databaseID)
}
