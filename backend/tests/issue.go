package tests

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1 "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) closeIssue(ctx context.Context, projectName, issueName string) error {
	if _, err := ctl.issueServiceClient.BatchUpdateIssuesStatus(ctx, &v1.BatchUpdateIssuesStatusRequest{
		Parent: projectName,
		Issues: []string{issueName},
		Status: v1.IssueStatus_DONE,
	}); err != nil {
		return err
	}
	return nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssue(id int) (*api.Issue, error) {
	body, err := ctl.get(fmt.Sprintf("/issue/%d", id), nil)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get issue response")
	}
	return issue, nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssues(projectID *int, statusList ...api.IssueStatus) ([]*api.Issue, error) {
	var ret []*api.Issue
	// call getOnePageIssuesWithToken until no more issues.
	token := ""
	for {
		issues, nextToken, err := ctl.getOnePageIssuesWithToken(projectID, statusList, token)
		if err != nil {
			return nil, err
		}
		if len(issues) == 0 {
			break
		}
		ret = append(ret, issues...)
		token = nextToken
	}
	return ret, nil
}

func (ctl *controller) getOnePageIssuesWithToken(projectID *int, statusList []api.IssueStatus, token string) ([]*api.Issue, string, error) {
	params := make(map[string]string)
	if projectID != nil {
		params["project"] = fmt.Sprintf("%d", *projectID)
	}
	if len(statusList) > 0 {
		var sl []string
		for _, status := range statusList {
			sl = append(sl, string(status))
		}
		params["status"] = strings.Join(sl, ",")
	}
	if token != "" {
		params["token"] = token
	}
	body, err := ctl.get("/issue", params)
	if err != nil {
		return nil, "", err
	}
	issueResp := new(api.IssueResponse)
	err = jsonapi.UnmarshalPayload(body, issueResp)
	if err != nil {
		return nil, "", errors.Wrap(err, "fail to unmarshal get issue response")
	}
	return issueResp.Issues, issueResp.NextToken, nil
}
