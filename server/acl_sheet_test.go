package server

import (
	"testing"

	"github.com/bytebase/bytebase/api"
)

func TestEnforceWorkspaceDeveloperSheetRouteACL(t *testing.T) {
	type test struct {
		desc        string
		plan        api.PlanType
		path        string
		method      string
		principalID int
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Fetch own sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/my",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch shared sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/shared",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Fetch starred sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/starred",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
	}

	privateSheetTests := []test{
		{
			desc:        "Access private sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Access private sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: 200,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Access private sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: 201,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Access private sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: 204,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Change private sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: 204,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Organize private sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000/organize",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Organize private sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000/organize",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Organize private sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000/organize",
			method:      "PATCH",
			principalID: 204,
			errMsg:      "not allowed to access private sheet created by other user",
		},
	}

	projectSheetTests := []test{
		{
			desc:        "Access project sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Access project sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Access project sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: 201,
			errMsg:      "",
		},
		{
			desc:        "Access project sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: 204,
			errMsg:      "is not a member of the project containing the sheet",
		},
		{
			desc:        "Change project sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Change project sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Change project sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "not have permission to change the project sheet",
		},
		{
			desc:        "Change project sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: 204,
			errMsg:      "is not a member of the project containing the sheet",
		},
		{
			desc:        "Organize project sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Organize project sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: 200,
			errMsg:      "",
		},
		{
			desc:        "Organize project sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: 201,
			errMsg:      "",
		},
		{
			desc:        "Organize project sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: 204,
			errMsg:      "is not a member of the project containing the sheet",
		},
	}

	publicSheetTests := []test{
		{
			desc:        "Access public sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "GET",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Access public sheet as a non-creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "GET",
			principalID: 204,
			errMsg:      "",
		},
		{
			desc:        "Change public sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Change public sheet as a non-creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "PATCH",
			principalID: 204,
			errMsg:      "not allowed to change public sheet created by other user",
		},
		{
			desc:        "Organize public sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002/organize",
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Organize public sheet as a non-creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002/organize",
			method:      "PATCH",
			principalID: 204,
			errMsg:      "",
		},
	}

	tests = append(tests, privateSheetTests...)
	tests = append(tests, projectSheetTests...)
	tests = append(tests, publicSheetTests...)

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperSheetRouteACL(tc.plan, tc.path, tc.method, tc.principalID, roleFinder, sheetFinder)
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
