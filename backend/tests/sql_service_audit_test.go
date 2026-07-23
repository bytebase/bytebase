package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestAdminExecuteAuditLog(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "admin execute audit",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)

	databaseName := "admin_execute_audit"
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, databaseName, ""))
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: instanceResp.Msg.Name + "/databases/" + databaseName,
	}))
	a.NoError(err)

	results, err := ctl.adminQuery(ctx, databaseResp.Msg, "SELECT 1")
	a.NoError(err)
	a.NotEmpty(results)

	logs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  ctl.project.Name,
		Filter:  `method == "/bytebase.v1.SQLService/AdminExecute"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	a.NotEmpty(logs.Msg.AuditLogs, "AdminExecute must produce an audit entry")
	entry := logs.Msg.AuditLogs[0]
	a.Equal("/bytebase.v1.SQLService/AdminExecute", entry.Method)
	a.Equal(databaseResp.Msg.Name, entry.Resource)
}
