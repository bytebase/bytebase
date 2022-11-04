package server

import (
	"net/url"
	"testing"

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
		case 202:
			return common.ProjectOwner, nil
		}
	}
	return "", nil
}

func TestEnforceWorkspaceDeveloperProjectRouteACL(t *testing.T) {
	type test struct {
		desc        string
		plan        api.PlanType
		path        string
		method      string
		queryParams url.Values
		principalID int
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Create a project",
			plan:        api.ENTERPRISE,
			path:        "/project",
			method:      "POST",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch all projects",
			plan:        api.ENTERPRISE,
			path:        "/project",
			method:      "GET",
			queryParams: url.Values{},
			principalID: 200,
			errMsg:      "not allowed to fetch all project list",
		},
		{
			desc:        "Fetch all projects from other user",
			plan:        api.ENTERPRISE,
			path:        "/project",
			method:      "GET",
			queryParams: url.Values{"user": []string{"201"}},
			principalID: 200,
			errMsg:      "not allowed to fetch projects from other user",
		},
		{
			desc:        "Fetch all projects from themselves",
			plan:        api.ENTERPRISE,
			path:        "/project",
			method:      "GET",
			queryParams: url.Values{"user": []string{"200"}},
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch a single project",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch a single project",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Patch a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Patch a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; PATCH /project/100 u201/DEVELOPER",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperProjectRouteACL(tc.plan, tc.path, tc.method, tc.queryParams, tc.principalID, roleFinder)
			if err != nil {
				if tc.errMsg == "" {
					t.Errorf("expect no error, got %s", err.Internal.Error())
				} else if tc.errMsg != err.Internal.Error() {
					t.Errorf("expect error %s, got %s", tc.errMsg, err.Internal.Error())
				}
			} else if tc.errMsg != "" {
				t.Errorf("expect error %s, got no error", tc.errMsg)
			}
		})
	}
}
