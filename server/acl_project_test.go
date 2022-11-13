package server

import (
	"net/url"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

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
			desc:        "Fetch a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Fetch a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "",
		},
		{
			desc:        "Fetch a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "",
		},
		{
			desc:        "Patch a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Patch a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project general setting",
		},
		{
			desc:        "Patch a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
		{
			desc:        "Fetch subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "GET",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Fetch subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "GET",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "",
		},
		{
			desc:        "Fetch subroute from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "GET",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "",
		},
		{
			desc:        "Create subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "POST",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Create subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "POST",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project general setting",
		},
		{
			desc:        "Create subroute from a single project as a non member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "POST",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
		{
			desc:        "Patch subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Patch subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project general setting",
		},
		{
			desc:        "Patch subroute from a single project as a non member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
		{
			desc:        "Delete subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "DELETE",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Delete subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "DELETE",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project general setting",
		},
		{
			desc:        "Delete subroute from a single project as a non member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "DELETE",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
		{
			desc:        "Create member from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member",
			method:      "POST",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "Create member from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member",
			method:      "POST",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project member",
		},
		{
			desc:        "Create member from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member",
			method:      "POST",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
		{
			desc:        "PATCH member from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "PATCH member from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project member",
		},
		{
			desc:        "PATCH member from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
		{
			desc:        "DELETE member from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "DELETE",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectOwner),
			errMsg:      "",
		},
		{
			desc:        "DELETE member from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "DELETE",
			principalID: testFindPrincipalIDFromProject(100, common.ProjectDeveloper),
			errMsg:      "not have permission to manage the project member",
		},
		{
			desc:        "DELETE member from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "DELETE",
			principalID: testFindPrincipalIDFromProject(100, ""),
			errMsg:      "is not a member of the project",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperProjectRouteACL(tc.plan, tc.path, tc.method, tc.queryParams, tc.principalID, roleFinder)
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
