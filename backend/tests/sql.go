package tests

import (
	"context"
	"time"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) adminQuery(ctx context.Context, database *v1pb.Database, query string) ([]*v1pb.QueryResult, error) {
	c, err := ctl.sqlServiceClient.AdminExecute(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.Send(&v1pb.AdminExecuteRequest{
		Name:      database.Name,
		Statement: query,
	}); err != nil {
		return nil, err
	}
	resp, err := c.Recv()
	if err != nil {
		return nil, err
	}
	if err := c.CloseSend(); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// GetSQLReviewResult will wait for next task SQL review task check to finish and return the task check result.
func (ctl *controller) GetSQLReviewResult(ctx context.Context, plan *v1pb.Plan) (*v1pb.PlanCheckRun, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := ctl.planServiceClient.ListPlanCheckRuns(ctx, &v1pb.ListPlanCheckRunsRequest{
			Parent: plan.Name,
		})
		if err != nil {
			return nil, err
		}
		for _, check := range resp.PlanCheckRuns {
			if check.Type == v1pb.PlanCheckRun_DATABASE_STATEMENT_ADVISE {
				if check.Status == v1pb.PlanCheckRun_DONE || check.Status == v1pb.PlanCheckRun_FAILED {
					return check, nil
				}
			}
		}
	}
	return nil, nil
}

// getSchemaDiff gets the schema diff.
func (ctl *controller) getSchemaDiff(ctx context.Context, schemaDiff *v1pb.DiffSchemaRequest) (string, error) {
	resp, err := ctl.databaseServiceClient.DiffSchema(ctx, schemaDiff)
	if err != nil {
		return "", err
	}
	return resp.Diff, nil
}
