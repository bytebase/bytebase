package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestSQLEditorTableScopedDML reproduces SUP-222: a project IAM grant that gives
// bb.sql.dml scoped to a specific table must authorize an INSERT to that table in
// the SQL Editor.
//
// Postgres is required: it is a "newACL" engine, so validateQueryRequest /
// ValidateSQLForEditor is skipped (backend/api/v1/sql_service.go:322). That skip
// is the only reason a DML statement reaches the ACL at all; on a non-newACL
// engine the INSERT would be rejected at validation for the wrong reason.
//
// The bug: the DML access check evaluates the grant's CEL condition at database
// granularity and never supplies resource.table_name (see checkDatabaseAccess in
// backend/api/v1/sql_service.go), so a resource.table_name-scoped condition can't
// match. A SELECT on the same table works because the SELECT path supplies
// per-column resource.table_name. Until the fix lands, the assertion below FAILS
// with "permission denied to access resources: <db>" — the expected RED outcome.
func TestSQLEditorTableScopedDML(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Save the owner token so we can swap identities and swap back.
	ownerToken := ctl.authInterceptor.token

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	// 1. Create a Postgres instance + database as the owner.
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "postgres", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	const databaseName = "sup222"
	err = ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "postgres")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	// 2. Create the table public.locked_accounts via the change-database flow.
	setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE public.locked_accounts (account_id bigint);`)},
	}))
	a.NoError(err)
	err = ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false)
	a.NoError(err)

	// 3. Create a custom role with only bb.sql.dml.
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role: &v1pb.Role{
			Title:       "repro-write",
			Permissions: []string{"bb.sql.dml"},
		},
		RoleId: "repro-write",
	}))
	a.NoError(err)

	// 4. Create a limited, non-owner user.
	limitedEmail := fmt.Sprintf("limited-%s@example.com", generateRandomString("u"))
	limitedPassword := "1024bytebase"
	limitedUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    limitedEmail,
			Password: limitedPassword,
			Title:    "Limited User",
		},
	}))
	a.NoError(err)

	// Add to workspace IAM so the user can login.
	_, err = ctl.addMemberToWorkspaceIAM(ctx, limitedUser.Msg.Workspace, fmt.Sprintf("user:%s", limitedEmail), "roles/workspaceMember")
	a.NoError(err)

	// 5. Set the project IAM policy: the limited user gets an unconditional
	// sqlEditorReadUser binding (method-gate + bb.sql.select, NO bb.sql.dml) plus
	// a table-scoped repro-write binding.
	envID, err := common.GetEnvironmentID(database.GetEffectiveEnvironment()) // bare id, e.g. "prod"
	a.NoError(err)
	condition := fmt.Sprintf(
		`resource.database == "%s/databases/%s" && resource.table_name in ["locked_accounts"] && resource.environment_id in ["%s"]`,
		instance.Name, databaseName, envID,
	)

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings,
		&v1pb.Binding{
			Role:    "roles/sqlEditorReadUser",
			Members: []string{fmt.Sprintf("user:%s", limitedEmail)},
		},
		&v1pb.Binding{
			Role:    "roles/repro-write",
			Members: []string{fmt.Sprintf("user:%s", limitedEmail)},
			Condition: &expr.Expr{
				Expression: condition,
			},
		},
	)
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policy,
	}))
	a.NoError(err)

	// 6. Swap identity to the limited user and run the INSERT.
	// Non-web login so the bearer token is returned in the response body
	// (Web:true delivers the token only as a Set-Cookie, which the test's
	// bearer-token interceptor does not read).
	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    limitedEmail,
		Password: limitedPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = loginResp.Msg.Token

	queryResp, queryErr := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "INSERT INTO public.locked_accounts (account_id) VALUES (1);",
		Limit:     1000,
	}))

	// 7. Assert the write was authorized and executed. The table-scoped bb.sql.dml
	// grant SHOULD authorize this INSERT. Until SUP-222 is fixed, the DML access
	// check denies it and surfaces "permission denied to access resources: <db>".
	a.NoError(queryErr, "INSERT should be authorized by the table-scoped bb.sql.dml grant")
	a.NotNil(queryResp)
	a.Len(queryResp.Msg.Results, 1)
	a.Empty(queryResp.Msg.Results[0].Error, "result should carry no error")
	a.Nil(queryResp.Msg.Results[0].GetPermissionDenied(), "result should carry no permission_denied detail")

	// Swap back to the owner and confirm the row actually landed.
	ctl.authInterceptor.token = ownerToken
	countResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT count(*) FROM public.locked_accounts;",
		Limit:     1,
	}))
	a.NoError(err)
	a.Len(countResp.Msg.Results, 1)
	a.Empty(countResp.Msg.Results[0].Error)
	a.Len(countResp.Msg.Results[0].Rows, 1)
	a.Equal(int64(1), countResp.Msg.Results[0].Rows[0].Values[0].GetInt64Value(), "the inserted row should be present")
}
