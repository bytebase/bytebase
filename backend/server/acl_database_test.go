package server

import (
	"testing"

	api "github.com/bytebase/bytebase/backend/legacyapi"

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
		plan        api.PlanType
	}

	tests := []test{
		// Retrieve the database which is unassigned to any project.
		{
			desc:        "Retrieve the database which is assigned to a project, and the user is a member of the project",
			path:        "/database/303",
			method:      "GET",
			body:        "",
			principalID: 201,
			errMsg:      "",
			plan:        api.ENTERPRISE,
		},

		// Transfer the database.
		{
			desc:        "Enterprise Plan, workspace developer of project developer cannot transfer the database out",
			path:        "/database/304",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":104}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "user is not project owner of project owns the database 304",
			plan:        api.ENTERPRISE,
		},
		{
			desc:        "Free & Pro Plan, workspace developer of project developer can transfer the database out",
			path:        "/database/304",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":104}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "",
			plan:        api.FREE,
		},
		{
			desc:        "Enterprise Plan, workspace developer of project owner cannot transfer the database out to the project if he is not the owner",
			path:        "/database/303",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":103}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "user is not project owner of project want owns the database 303",
			plan:        api.ENTERPRISE,
		},
		{
			desc:        "Free & Pro Plan, workspace developer of project owner can transfer the database out to the project if he is not the owner",
			path:        "/database/303",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":103}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "",
			plan:        api.TEAM,
		},
		{
			desc:        "Transfer the database to the project which the user is not a member of",
			path:        "/database/303",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":101}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "user is not a project member of project want to owns the database 303",
			plan:        api.FREE,
		},
		{
			desc:        "Transfer out the database under the project which the user is not a member of",
			path:        "/database/302",
			body:        `{"data":{"type":"databasePatch","attributes":{"projectId":102}}}`,
			method:      "PATCH",
			principalID: 201,
			errMsg:      "user is not a project member of project owns the database 302",
			plan:        api.FREE,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperDatabaseRouteACL(tc.plan, tc.path, tc.method, tc.body, tc.principalID, testWorkspaceDeveloperDatabaseRouteMockGetDatabaseProjectID, getProjectRolesFinderForTest(testWorkspaceDeveloperDatabaseRouteHelper.projectMembers))
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
			common.ProjectOwner: {201},
		},
		103: {
			common.ProjectDeveloper: {201},
		},
		104: {
			common.ProjectOwner: {201},
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
		104: {},
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
