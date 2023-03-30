package server

import (
	"net/url"
	"testing"
)

func TestWorkspaceDeveloperIssueRouteACL_RetrieveIssue(t *testing.T) {
	type test struct {
		desc        string
		path        string
		method      string
		queryParams url.Values
		principalID int
		body        string
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Retrieve other users' issue",
			path:        "/issue?user=201",
			queryParams: url.Values{"user": []string{"201"}},
			method:      "GET",
			principalID: 202,
			errMsg:      "not allowed to list other users' issues",
		},
		{
			desc:        "Retrieve my issue",
			path:        "/issue?user=202",
			queryParams: url.Values{"user": []string{"202"}},
			method:      "GET",
			principalID: 202,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperIssueRouteACL(tc.path, tc.method, tc.body, tc.queryParams, tc.principalID, testWorkspaceDeveloperIssueRouteMockGetIssueProjectID, testWorkspaceDeveloperIssueRouteMockGetProjectMemberIDs)
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
		path        string
		method      string
		body        string
		queryParams url.Values
		principalID int
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Operating the issue created by other user in other projects",
			path:        "/issue/403/status",
			body:        "",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 202,
			errMsg:      "not allowed to operate the issue",
		},
		{
			desc:        "Operating the issue I created",
			path:        "/issue/401/status",
			body:        "",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
		{
			desc:        "Operating the issue created by other user in projects I am a member of",
			path:        "/issue/402/status",
			body:        "",
			queryParams: url.Values{},
			method:      "PATCH",
			principalID: 202,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperIssueRouteACL(tc.path, tc.method, tc.body, tc.queryParams, tc.principalID, testWorkspaceDeveloperIssueRouteMockGetIssueProjectID, testWorkspaceDeveloperIssueRouteMockGetProjectMemberIDs)
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
		path        string
		method      string
		body        string
		queryParams url.Values
		principalID int
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Create issue under project I am not a member of",
			path:        "/issue",
			body:        `{"data":{"type":"issue","attributes":{"title":"test","description":"test","projectId":103}}}`,
			queryParams: url.Values{},
			method:      "POST",
			principalID: 202,
			errMsg:      "not allowed to create issue under the project 103",
		},
		{
			desc:        "Create issue under project I am a member of",
			path:        "/issue",
			body:        `{"data":{"type":"issue","attributes":{"title":"test","description":"test","projectId":102}}}`,
			queryParams: url.Values{},
			method:      "POST",
			principalID: 202,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperIssueRouteACL(tc.path, tc.method, tc.body, tc.queryParams, tc.principalID, testWorkspaceDeveloperIssueRouteMockGetIssueProjectID, testWorkspaceDeveloperIssueRouteMockGetProjectMemberIDs)
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
