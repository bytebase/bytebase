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
			desc:        "Fetch a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: 201,
			errMsg:      "",
		},
		{
			desc:        "Fetch a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "GET",
			principalID: 202,
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
		{
			desc:        "Patch a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; PATCH /project/100 u202/non-member",
		},
		{
			desc:        "Fetch subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "GET",
			principalID: 201,
			errMsg:      "",
		},
		{
			desc:        "Fetch subroute from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "GET",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Create subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "POST",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Create subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "POST",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; POST /project/100/repository u201/DEVELOPER",
		},
		{
			desc:        "Create subroute from a single project as a non member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "POST",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; POST /project/100/repository u202/non-member",
		},
		{
			desc:        "Patch subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Patch subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; PATCH /project/100/repository u201/DEVELOPER",
		},
		{
			desc:        "Patch subroute from a single project as a non member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; PATCH /project/100/repository u202/non-member",
		},
		{
			desc:        "Delete subroute from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "DELETE",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Delete subroute from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "DELETE",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; DELETE /project/100/repository u201/DEVELOPER",
		},
		{
			desc:        "Delete subroute from a single project as a non member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/repository",
			method:      "DELETE",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; DELETE /project/100/repository u202/non-member",
		},
		{
			desc:        "Create member from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member",
			method:      "POST",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Create member from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member",
			method:      "POST",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; POST /project/100/member u201/DEVELOPER",
		},
		{
			desc:        "Create member from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member",
			method:      "POST",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; POST /project/100/member u202/non-member",
		},
		{
			desc:        "PATCH member from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "PATCH member from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; PATCH /project/100/member/567 u201/DEVELOPER",
		},
		{
			desc:        "PATCH member from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; PATCH /project/100/member/567 u202/non-member",
		},
		{
			desc:        "DELETE member from a single project as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "DELETE",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "DELETE member from a single project as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "DELETE",
			principalID: 201,
			errMsg:      "rejected by the project ACL policy; DELETE /project/100/member/567 u201/DEVELOPER",
		},
		{
			desc:        "DELETE member from a single project as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/project/100/member/567",
			method:      "DELETE",
			principalID: 202,
			errMsg:      "rejected by the project ACL policy; DELETE /project/100/member/567 u202/non-member",
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
