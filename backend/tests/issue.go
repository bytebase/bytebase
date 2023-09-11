package tests

import (
	"context"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) getLastOpenIssue(ctx context.Context, project *v1pb.Project) (*v1pb.Issue, error) {
	resp, err := ctl.issueServiceClient.ListIssues(ctx, &v1pb.ListIssuesRequest{
		Parent: project.Name,
		Filter: "status = OPEN",
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Issues) == 0 {
		return nil, nil
	}
	return resp.Issues[0], nil
}

func (ctl *controller) closeIssue(ctx context.Context, project *v1pb.Project, issueName string) error {
	if _, err := ctl.issueServiceClient.BatchUpdateIssuesStatus(ctx, &v1pb.BatchUpdateIssuesStatusRequest{
		Parent: project.Name,
		Issues: []string{issueName},
		Status: v1pb.IssueStatus_DONE,
	}); err != nil {
		return err
	}
	return nil
}
