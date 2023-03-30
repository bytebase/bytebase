package server

import (
	"testing"
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
			err := enforceWorkspaceDeveloperDatabaseRouteACL(tc.path, tc.method, tc.body, tc.principalID, testWorkspaceDeveloperDatabaseRouteMockGetDatabaseProjectID, testWorkspaceDeveloperDatabaseRouteMockGetProjectMemberIDs)
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
