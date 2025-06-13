package tests

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) adminQuery(ctx context.Context, database *v1pb.Database, query string) ([]*v1pb.QueryResult, error) {
	stream := ctl.sqlServiceClient.AdminExecute(ctx)
	if err := stream.Send(&v1pb.AdminExecuteRequest{
		Name:      database.Name,
		Statement: query,
	}); err != nil {
		return nil, err
	}
	resp, err := stream.Receive()
	if err != nil {
		return nil, err
	}
	if err := stream.CloseRequest(); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// GetSQLReviewResult will wait for next task SQL review task check to finish and return the task check result.
func (ctl *controller) GetSQLReviewResult(ctx context.Context, plan *v1pb.Plan) (*v1pb.PlanCheckRun, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := ctl.planServiceClient.ListPlanCheckRuns(ctx, connect.NewRequest(&v1pb.ListPlanCheckRunsRequest{
			Parent: plan.Name,
		}))
		if err != nil {
			return nil, err
		}
		for _, check := range resp.Msg.PlanCheckRuns {
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
	resp, err := ctl.databaseServiceClient.DiffSchema(ctx, connect.NewRequest(schemaDiff))
	if err != nil {
		return "", err
	}
	return resp.Msg.Diff, nil
}
