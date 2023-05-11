package server

import (
	"net/url"
	"testing"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func TestWorkspaceDeveloperIssueRouteACL_RetrieveIssue(t *testing.T) {
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
			desc:        "Retrieve other users' issue by project owner",
			plan:        api.ENTERPRISE,
			path:        "/issue?user=201",
			queryParams: url.Values{"user": []string{"201"}},
			method:      "GET",
			principalID: 202,
			errMsg:      "not allowed to list other users' issues",
		},
		{
			desc:        "Retrieve other users' issue by custom role",
			plan:        api.ENTERPRISE,
			path:        "/issue?user=201",
			queryParams: url.Values{"user": []string{"201"}},
			method:      "GET",
			principalID: 204,
			errMsg:      "not allowed to list other users' issues",
		},
		{
			desc:        "Retrieve my issue",
			plan:        api.ENTERPRISE,
			path:        "/issue?user=202",
			queryParams: url.Values{"user": []string{"202"}},
			method:      "GET",
			principalID: 202,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperIssueRouteACL(tc.plan, tc.path, tc.method, tc.queryParams, tc.principalID, testWorkspaceDeveloperIssueRouteMockGetIssueProjectID, getProjectRolesFinderForTest(testWorkspaceDeveloperIssueRouteHelper.projectMembers))
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

func TestWorkspaceDeveloperIssueRouteACL_OperateIssue(t *testing.T) {
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
			desc:        "Operating the issue created by other user in other projects",
			plan:        api.ENTERPRISE,
			path:        "/issue/403/status",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 202,
			errMsg:      "not allowed to operate the issue",
		},
		{
			desc:        "Operating the issue I created",
			plan:        api.ENTERPRISE,
			path:        "/issue/401/status",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Operating the issue created by other user in projects I am a member of",
			plan:        api.ENTERPRISE,
			path:        "/issue/402/status",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Operating the issue created by other user in projects I am a member of but I'm neither a project owner nor a developer",
			plan:        api.ENTERPRISE,
			path:        "/issue/402/status",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 204,
			errMsg:      "not allowed to operate the issue",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperIssueRouteACL(tc.plan, tc.path, tc.method, tc.queryParams, tc.principalID, testWorkspaceDeveloperIssueRouteMockGetIssueProjectID, getProjectRolesFinderForTest(testWorkspaceDeveloperIssueRouteHelper.projectMembers))
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

func TestWorkspaceDeveloperIssueRouteACL_CreateIssue(t *testing.T) {
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
			desc:        "Create issue under project I am a member of",
			plan:        api.ENTERPRISE,
			path:        "/issue",
			queryParams: url.Values{},
			method:      "POST",
			principalID: 202,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperIssueRouteACL(tc.plan, tc.path, tc.method, tc.queryParams, tc.principalID, testWorkspaceDeveloperIssueRouteMockGetIssueProjectID, getProjectRolesFinderForTest(testWorkspaceDeveloperIssueRouteHelper.projectMembers))
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

var testWorkspaceDeveloperIssueRouteHelper = struct {
	projectMembers map[int]map[common.ProjectRole][]int
	// issueIDToProjectID is a map from issue ID to the project ID.
	issueIDToProjectID map[int]int
}{
	projectMembers: map[int]map[common.ProjectRole][]int{
		// Project 102 contains members 202 and 203.
		102: {
			common.ProjectOwner:              {202},
			common.ProjectDeveloper:          {202, 203},
			common.ProjectRole("CustomRole"): {204},
		},
		// Project 103 contains member 204.
		103: {
			common.ProjectOwner: {204},
		},
	},
	issueIDToProjectID: map[int]int{
		// Issue 401 belongs to project 102.
		401: 102,
		// Issue 402 belongs to project 102.
		402: 102,
		// Issue 403 belongs to project 103.
		403: 103,
	},
}

func testWorkspaceDeveloperIssueRouteMockGetIssueProjectID(issueID int) (int, error) {
	projectID, ok := testWorkspaceDeveloperIssueRouteHelper.issueIDToProjectID[issueID]
	if !ok {
		return 0, errors.Errorf("issue %d does not belong to any project", issueID)
	}
	return projectID, nil
}
