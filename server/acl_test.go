package server

import (
	"net/url"
	"testing"
)

func TestEnforceWorkspaceDeveloperProjectACL(t *testing.T) {
	type test struct {
		desc        string
		path        string
		method      string
		queryParams url.Values
		principalID int
		errMsg      string
	}

	tests := []test{
		{
			desc:        "Fetch all projects",
			path:        "/project",
			method:      "GET",
			queryParams: url.Values{},
			principalID: 123,
			errMsg:      "Not allowed to fetch all project list",
		},
		{
			desc:        "Fetch projects from other user",
			path:        "/project",
			method:      "GET",
			queryParams: url.Values{"user": []string{"124"}},
			principalID: 123,
			errMsg:      "Not allowed to fetch projects from other user",
		},
		{
			desc:        "Fetch projects from herself",
			path:        "/project",
			method:      "GET",
			queryParams: url.Values{"user": []string{"123"}},
			principalID: 123,
			errMsg:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := enforceWorkspaceDeveloperProjectACL(tc.path, tc.method, tc.queryParams, tc.principalID)
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
