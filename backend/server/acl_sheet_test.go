package server

import (
	"testing"

	api "github.com/bytebase/bytebase/backend/legacyapi"
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
			principalID: 1234,
			errMsg:      "",
		},
		{
			desc:        "Fetch shared sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/shared",
			method:      "GET",
			principalID: 1234,
			errMsg:      "",
		},
		{
			desc:        "Fetch starred sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/starred",
			method:      "GET",
			principalID: 1234,
			errMsg:      "",
		},
		{
			desc:        "Access unknown sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/9999",
			method:      "GET",
			principalID: 1234,
			errMsg:      "Sheet ID not found: 9999",
		},
		{
			desc:        "Organize unknown sheet",
			plan:        api.ENTERPRISE,
			path:        "/sheet/9999/organize",
			method:      "PATCH",
			principalID: 1234,
			errMsg:      "Sheet ID not found: 9999",
		},
	}

	privateSheetTests := []test{
		{
			desc:        "Access private sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1000, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Access private sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1000, "OWNER"),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Access private sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1000, "DEVELOPER"),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Access private sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1000, ""),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Change private sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, "OWNER"),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, "DEVELOPER"),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, ""),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Change private sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Organize private sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, "OWNER"),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Organize private sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, "DEVELOPER"),
			errMsg:      "not allowed to access private sheet created by other user",
		},
		{
			desc:        "Organize private sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1000/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1000, ""),
			errMsg:      "not allowed to access private sheet created by other user",
		},
	}

	projectSheetTests := []test{
		{
			desc:        "Access project sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1001, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Access project sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1001, "OWNER"),
			errMsg:      "",
		},
		{
			desc:        "Access project sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1001, "DEVELOPER"),
			errMsg:      "",
		},
		{
			desc:        "Access project sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1001, ""),
			errMsg:      "is not a member of the project containing the sheet",
		},
		{
			desc:        "Change project sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Change project sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, "OWNER"),
			errMsg:      "",
		},
		{
			desc:        "Change project sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, "DEVELOPER"),
			errMsg:      "not have permission to change the project sheet",
		},
		{
			desc:        "Change project sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, ""),
			errMsg:      "is not a member of the project containing the sheet",
		},
		{
			desc:        "Organize project sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Organize project sheet as a project owner",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, "OWNER"),
			errMsg:      "",
		},
		{
			desc:        "Organize project sheet as a project developer",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, "DEVELOPER"),
			errMsg:      "",
		},
		{
			desc:        "Organize project sheet as a non-member",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1001/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1001, ""),
			errMsg:      "is not a member of the project containing the sheet",
		},
	}

	publicSheetTests := []test{
		{
			desc:        "Access public sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1002, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Access public sheet as a non-creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "GET",
			principalID: testFindPrincipalIDFromSheet(1002, ""),
			errMsg:      "",
		},
		{
			desc:        "Change public sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1002, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Change public sheet as a non-creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1002, ""),
			errMsg:      "not allowed to change public sheet created by other user",
		},
		{
			desc:        "Organize public sheet as a creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1002, "CREATOR"),
			errMsg:      "",
		},
		{
			desc:        "Organize public sheet as a non-creator",
			plan:        api.ENTERPRISE,
			path:        "/sheet/1002/organize",
			method:      "PATCH",
			principalID: testFindPrincipalIDFromSheet(1002, ""),
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
