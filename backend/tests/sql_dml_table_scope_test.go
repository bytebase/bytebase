package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
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

// TestSQLEditorTableScopedDMLEdgeCases exercises the regression + edge cases for
// the SUP-222 fix: DML/DDL grants are authorized per write-target table (via
// ExtractChangedResources), fall back to a database-level check when targets
// can't be resolved (never fail open), and are authorized per statement in
// multi-statement batches.
//
// All cases below MUST pass: they assert the fix's correctness and its edges.
// One server/instance/database is shared across cases via subtests; each subtest
// uses a fresh limited user + a per-case project IAM policy so grants don't leak.
//
// Postgres is required for the same reason the base test documents: it is a
// "newACL" engine, so DML/DDL statements reach the ACL at all.
func TestSQLEditorTableScopedDMLEdgeCases(t *testing.T) {
	// Not t.Parallel(): the subtests share one server/instance/database and each
	// rewrites the project IAM policy, so they must run serially (not in parallel
	// with each other). The whole test still runs alongside other test binaries.
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	// Single Postgres instance + database shared by every subtest.
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

	const databaseName = "sup222edge"
	a.NoError(ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, "postgres"))

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	envID, err := common.GetEnvironmentID(database.GetEffectiveEnvironment()) // bare id, e.g. "prod"
	a.NoError(err)

	// Create every table the subtests reference, plus a non-public schema, via the
	// change-database flow. Doing it once keeps the (slow) rollout flow off the hot path.
	setupSQL := strings.Join([]string{
		`CREATE TABLE public.t_granted (id bigint);`,
		`CREATE TABLE public.t_other (id bigint);`,
		`CREATE TABLE public.t_dst (id bigint);`,
		`CREATE TABLE public.t_src (id bigint);`,
		`CREATE TABLE public.a (id bigint);`,
		`CREATE TABLE public.b (id bigint);`,
		`CREATE SCHEMA other_schema;`,
		`CREATE TABLE other_schema.t_in_other (id bigint);`,
		`INSERT INTO public.t_src (id) VALUES (7), (8), (9);`,
	}, "\n")
	setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(setupSQL)},
	}))
	a.NoError(err)
	a.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

	// Custom roles used across cases. bb.sql.dml for writes, bb.sql.ddl for schema
	// changes — each table-scopable via the binding's CEL condition.
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role:   &v1pb.Role{Title: "repro-write", Permissions: []string{"bb.sql.dml"}},
		RoleId: "repro-write",
	}))
	a.NoError(err)
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		Role:   &v1pb.Role{Title: "repro-ddl", Permissions: []string{"bb.sql.ddl"}},
		RoleId: "repro-ddl",
	}))
	a.NoError(err)

	// Capture the baseline project IAM policy so each subtest can reset to it
	// before layering on its own bindings (no cross-subtest grant leakage).
	baselineResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	baselineBindings := baselineResp.Msg.Bindings

	dbFullName := fmt.Sprintf("%s/databases/%s", instance.Name, databaseName)

	// newLimitedUser creates a fresh non-owner user that can log in, and returns its
	// email + a bearer token. Run as the owner before calling.
	newLimitedUser := func(t *testing.T) (string, string) {
		t.Helper()
		ra := require.New(t)
		ctl.authInterceptor.token = ownerToken
		email := fmt.Sprintf("limited-%s@example.com", generateRandomString("u"))
		password := "1024bytebase"
		u, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
			User: &v1pb.User{Email: email, Password: password, Title: "Limited User"},
		}))
		ra.NoError(err)
		_, err = ctl.addMemberToWorkspaceIAM(ctx, u.Msg.Workspace, fmt.Sprintf("user:%s", email), "roles/workspaceMember")
		ra.NoError(err)
		loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: password}))
		ra.NoError(err)
		return email, loginResp.Msg.Token
	}

	// setProjectBindings resets the project IAM policy to baseline + the given
	// extra bindings. Run as the owner.
	setProjectBindings := func(t *testing.T, extra ...*v1pb.Binding) {
		t.Helper()
		ra := require.New(t)
		ctl.authInterceptor.token = ownerToken
		policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
			Resource: ctl.project.Name,
		}))
		ra.NoError(err)
		policy := policyResp.Msg
		policy.Bindings = append(append([]*v1pb.Binding{}, baselineBindings...), extra...)
		_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
			Resource: ctl.project.Name,
			Policy:   policy,
		}))
		ra.NoError(err)
	}

	// readBinding is the unconditional sqlEditorReadUser binding every write case
	// pairs with (method-gate + bb.sql.select, no bb.sql.dml/ddl).
	readBinding := func(email string) *v1pb.Binding {
		return &v1pb.Binding{
			Role:    "roles/sqlEditorReadUser",
			Members: []string{fmt.Sprintf("user:%s", email)},
		}
	}
	// scopedBinding builds a conditional binding for the given role/condition.
	scopedBinding := func(role, email, condition string) *v1pb.Binding {
		return &v1pb.Binding{
			Role:      role,
			Members:   []string{fmt.Sprintf("user:%s", email)},
			Condition: &expr.Expr{Expression: condition},
		}
	}

	// runAs swaps to the user token, runs a Query, then restores the owner token.
	runAs := func(token, statement string) (*v1pb.QueryResponse, error) {
		ctl.authInterceptor.token = token
		resp, qErr := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: statement,
			Limit:     1000,
		}))
		ctl.authInterceptor.token = ownerToken
		if resp != nil {
			return resp.Msg, qErr
		}
		return nil, qErr
	}

	// assertAllowed asserts the Query was authorized: no error and no
	// permission_denied detail on any result.
	assertAllowed := func(t *testing.T, resp *v1pb.QueryResponse, qErr error) {
		t.Helper()
		ra := require.New(t)
		ra.NoError(qErr, "statement should be authorized")
		ra.NotNil(resp)
		ra.NotEmpty(resp.Results, "expected at least one result")
		for _, r := range resp.Results {
			ra.Empty(r.Error, "result should carry no error")
			ra.Nil(r.GetPermissionDenied(), "result should carry no permission_denied detail")
		}
	}

	// assertDeniedOn asserts the Query was denied on a specific resource. The denial
	// is surfaced on the last result's permission_denied detail (not as a top-level
	// error): queryRetryStopOnError attaches the access-check failure to the result's
	// Error field and returns a nil top-level error. The result detail must name a
	// resource containing wantResourceSubstr (e.g. ".../tables/t_other"), and no row
	// must have been written (rows == 0).
	assertDeniedOn := func(t *testing.T, resp *v1pb.QueryResponse, _ error, wantResourceSubstr string) {
		t.Helper()
		ra := require.New(t)
		ra.NotNil(resp)
		ra.NotEmpty(resp.Results)
		last := resp.Results[len(resp.Results)-1]
		pd := last.GetPermissionDenied()
		ra.NotNil(pd, "last result should carry a permission_denied detail; got error=%q", last.Error)
		ra.Empty(last.Rows, "a denied statement must not return/execute rows")
		joined := strings.Join(pd.Resources, ",")
		ra.Contains(joined, wantResourceSubstr, "denied resources %v should name %q", pd.Resources, wantResourceSubstr)
	}

	// countRows returns the row count of a table, queried as the owner.
	countRows := func(t *testing.T, table string) int64 {
		t.Helper()
		ra := require.New(t)
		ctl.authInterceptor.token = ownerToken
		resp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: fmt.Sprintf("SELECT count(*) FROM %s;", table),
			Limit:     1,
		}))
		ra.NoError(err)
		ra.Len(resp.Msg.Results, 1)
		ra.Empty(resp.Msg.Results[0].Error)
		ra.Len(resp.Msg.Results[0].Rows, 1)
		return resp.Msg.Results[0].Rows[0].Values[0].GetInt64Value()
	}

	tableScopedDML := func(email, table string) *v1pb.Binding {
		cond := fmt.Sprintf(
			`resource.database == "%s" && resource.table_name in ["%s"] && resource.environment_id in ["%s"]`,
			dbFullName, table, envID,
		)
		return scopedBinding("roles/repro-write", email, cond)
	}

	// 1. Denied on a non-granted table: grant is table-scoped to t_granted; an
	//    INSERT into t_other must be denied and name t_other.
	t.Run("DeniedOnNonGrantedTable", func(t *testing.T) {
		email, token := newLimitedUser(t)
		setProjectBindings(t, readBinding(email), tableScopedDML(email, "t_granted"))

		resp, qErr := runAs(token, "INSERT INTO public.t_other (id) VALUES (1);")
		assertDeniedOn(t, resp, qErr, "/tables/t_other")
	})

	// 2. Schema-scoped grant: condition scopes by schema_name only (no table_name).
	//    An INSERT into a public table is allowed AND executed; an INSERT into a
	//    table in a different schema is denied.
	t.Run("SchemaScopedGrant", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(
			`resource.database == "%s" && resource.schema_name == "public" && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		before := countRows(t, "public.t_dst")
		resp, qErr := runAs(token, "INSERT INTO public.t_dst (id) VALUES (101);")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_dst"), "the row should have been inserted")

		// A table outside the granted schema is denied.
		resp, qErr = runAs(token, "INSERT INTO other_schema.t_in_other (id) VALUES (1);")
		assertDeniedOn(t, resp, qErr, "other_schema/tables/t_in_other")
	})

	// 3. INSERT ... SELECT requires DML only on the target. The grant is table-scoped
	//    to t_dst; there is NO grant of any kind referencing t_src. The statement
	//    reads from t_src and writes to t_dst → allowed AND executed, proving
	//    read-source tables are not gated.
	t.Run("InsertSelectGatesTargetOnly", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		setProjectBindings(t, readBinding(email), tableScopedDML(email, "t_dst"))

		before := countRows(t, "public.t_dst")
		srcCount := countRows(t, "public.t_src")
		resp, qErr := runAs(token, "INSERT INTO public.t_dst SELECT id FROM public.t_src;")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+srcCount, countRows(t, "public.t_dst"), "all source rows should have been copied")
	})

	// 4. Database-scoped DML still works (no regression). The condition scopes by
	//    database + environment only (no table/schema clause).
	t.Run("DatabaseScopedDMLNoRegression", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(
			`resource.database == "%s" && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		before := countRows(t, "public.t_other")
		resp, qErr := runAs(token, "INSERT INTO public.t_other (id) VALUES (202);")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_other"), "the row should have been inserted")
	})

	// 5. UPDATE and DELETE on the granted table are allowed; the same statements
	//    against a non-granted table are denied.
	t.Run("UpdateDeleteScoped", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		setProjectBindings(t, readBinding(email), tableScopedDML(email, "t_granted"))

		// Seed a row to mutate (as the owner).
		ctl.authInterceptor.token = ownerToken
		_, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: "INSERT INTO public.t_granted (id) VALUES (1);",
			Limit:     1,
		}))
		ra.NoError(err)

		resp, qErr := runAs(token, "UPDATE public.t_granted SET id = 2 WHERE id = 1;")
		assertAllowed(t, resp, qErr)
		resp, qErr = runAs(token, "DELETE FROM public.t_granted WHERE id = 2;")
		assertAllowed(t, resp, qErr)

		// The same statements against t_other are denied.
		resp, qErr = runAs(token, "UPDATE public.t_other SET id = 2 WHERE id = 1;")
		assertDeniedOn(t, resp, qErr, "/tables/t_other")
		resp, qErr = runAs(token, "DELETE FROM public.t_other WHERE id = 1;")
		assertDeniedOn(t, resp, qErr, "/tables/t_other")
	})

	// 6. Fail-safe is closed: a user with ONLY the unconditional read pairing and no
	//    write grant must be denied (the no-matching-grant fallback denies, not grants).
	t.Run("FailSafeClosed", func(t *testing.T) {
		email, token := newLimitedUser(t)
		setProjectBindings(t, readBinding(email))

		resp, qErr := runAs(token, "INSERT INTO public.t_granted (id) VALUES (1);")
		assertDeniedOn(t, resp, qErr, "/tables/t_granted")
	})

	// 7. Multi-statement batches fall back to the database-level check (fail-closed). A
	//    batch can change the session's default schema mid-batch (Postgres SET search_path,
	//    Oracle ALTER SESSION SET CURRENT_SCHEMA), redirecting a later unqualified write to
	//    a schema the per-statement resolver can't reliably track. Rather than risk
	//    authorizing the wrong schema, multi-statement DML/DDL falls back to the
	//    database-level check — table/schema-scoped grants then require a database-level
	//    grant. Single statements stay per-write-target. SUP-222 / BYT-9698.
	t.Run("MultiStatementBatchFallsBackToDatabaseLevel", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		ddlCond := fmt.Sprintf(
			`resource.database == "%s" && resource.table_name in ["a"] && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		dmlCond := fmt.Sprintf(
			`resource.database == "%s" && resource.table_name in ["b"] && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t,
			readBinding(email),
			scopedBinding("roles/repro-ddl", email, ddlCond),
			scopedBinding("roles/repro-write", email, dmlCond),
		)

		// A 2-statement batch (DDL on a + DML on b), even though each table is individually
		// granted, is DENIED at the database level — table-scoped grants don't satisfy the
		// fallback. The denial names the bare database, NOT a /tables/ resource.
		col := "c" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		stmt := fmt.Sprintf("ALTER TABLE public.a ADD COLUMN %s int; INSERT INTO public.b (id) VALUES (1);", col)
		resp, qErr := runAs(token, stmt)
		assertDeniedOn(t, resp, qErr, dbFullName)
		last := resp.Results[len(resp.Results)-1]
		ra.NotContains(strings.Join(last.GetPermissionDenied().GetResources(), ","), "/tables/",
			"a multi-statement batch must deny via the database-level fallback, not a per-target check")

		// Sharpness: SINGLE statements are still authorized per write-target table — the DDL
		// grant on a does not authorize a DML on a, and the DML grant on b does not authorize
		// DDL on b.
		resp, qErr = runAs(token, "INSERT INTO public.a (id) VALUES (1);")
		assertDeniedOn(t, resp, qErr, "public/tables/a")
		col2 := "c" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		resp, qErr = runAs(token, fmt.Sprintf("ALTER TABLE public.b ADD COLUMN %s int;", col2))
		assertDeniedOn(t, resp, qErr, "public/tables/b")

		// Positive: with a DATABASE-scoped DML grant the same kind of multi-statement DML
		// batch is allowed (the fallback passes) — confirming the fallback is not a blanket
		// deny.
		dbDmlCond := fmt.Sprintf(`resource.database == "%s" && resource.environment_id in ["%s"]`, dbFullName, envID)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, dbDmlCond))
		before := countRows(t, "public.b")
		resp, qErr = runAs(token, "INSERT INTO public.b (id) VALUES (1); INSERT INTO public.b (id) VALUES (2);")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+2, countRows(t, "public.b"), "both INSERTs should execute under the database-level grant")
	})

	// 8. Empty-target fail-safe fallback (SUP-222). The DML/DDL access check resolves
	//    this statement's write targets via ExtractChangedResources; when that yields
	//    ZERO table targets it falls back to checkDatabaseAccess, which must stay
	//    fail-CLOSED. resolveWriteTargets returns zero targets in two converging cases:
	//      (a) a newACL engine WITHOUT a ChangedResources extractor — BigQuery,
	//          Snowflake, Spanner, MongoDB, Elasticsearch; and
	//      (b) DDL on a non-table object — ExtractChangedResources tracks tables only.
	//    Both hit the SAME engine-agnostic `len(targets)==0 → checkDatabaseAccess(perm)`
	//    guard in accessCheckWithGrantedTargets. Standing up BigQuery/Snowflake in the
	//    harness is expensive, so we exercise the IDENTICAL guard on Postgres with a
	//    non-table DDL — CREATE SEQUENCE — which parses to *ast.CreateSeqStmt: it
	//    classifies as DDL (perm = bb.sql.ddl) but is NOT handled by the pg
	//    ExtractChangedResources switch (which only adds tables for CREATE TABLE / DROP /
	//    ALTER TABLE / RENAME / CREATE INDEX / INSERT / UPDATE / DELETE), so it resolves
	//    to zero targets. This Postgres case therefore stands in for the
	//    unsupported-engine (no-extractor) case: same guard, same fail-closed contract.
	t.Run("EmptyTargetFailSafeFallback", func(t *testing.T) {
		// Unique sequence name so re-runs (and the two sub-cases) never collide.
		seqName := "s_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		createSeq := fmt.Sprintf("CREATE SEQUENCE public.%s;", seqName)
		ddlCond := func(extra string) string {
			return fmt.Sprintf(
				`resource.database == "%s"%s && resource.environment_id in ["%s"]`,
				dbFullName, extra, envID,
			)
		}

		// 8a. Db-scoped DDL grant → the empty-target fallback ALLOWS. The condition
		//     scopes by database + environment only (no table/schema clause), exactly
		//     the attributes checkDatabaseAccess supplies, so the fallback matches and
		//     the CREATE SEQUENCE executes.
		t.Run("DbScopedAllows", func(t *testing.T) {
			ra := require.New(t)
			email, token := newLimitedUser(t)
			setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-ddl", email, ddlCond("")))

			resp, qErr := runAs(token, createSeq)
			assertAllowed(t, resp, qErr)
			// Confirm the sequence actually landed (the fallback authorized execution).
			ra.Equal(int64(1), countRows(t,
				fmt.Sprintf("pg_sequences WHERE schemaname = 'public' AND sequencename = '%s'", seqName)),
				"the sequence should have been created")
		})

		// 8b. Table-scoped DDL grant → the empty-target fallback DENIES (never fail
		//     open). Same CREATE SEQUENCE, but the grant adds a resource.table_name
		//     clause. Because the statement has ZERO table targets, the code falls back
		//     to checkDatabaseAccess, whose attributes carry NO resource.table_name, so
		//     the table-scoped condition cannot match → denied at the database level.
		//     The db-level denial (resource = the bare database, not a /tables/… path)
		//     is itself the proof that targets were empty: had a non-empty target been
		//     resolved and matched the granted table, the statement would be ALLOWED.
		t.Run("TableScopedDenies", func(t *testing.T) {
			email, token := newLimitedUser(t)
			tableClause := ` && resource.table_name in ["any_table"]`
			setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-ddl", email, ddlCond(tableClause)))

			resp, qErr := runAs(token, createSeq)
			// Denied on the bare database (the fallback's checkDatabaseAccess resource),
			// NOT on a /tables/… path — confirming the empty-target path was taken.
			assertDeniedOn(t, resp, qErr, dbFullName)
			// assertDeniedOn does a substring Contains on dbFullName, which is a PREFIX of
			// any per-target resource name (instances/…/databases/…/schemas/…/tables/…), so
			// it alone cannot distinguish the bare-db fallback from a per-target denial.
			// Assert the denied resources carry NO /tables/ segment to PROVE the empty-target
			// fallback (checkDatabaseAccess) ran — guarding against a future regression where
			// resolveWriteTargets returns a spurious target.
			last := resp.Results[len(resp.Results)-1]
			require.New(t).NotContains(
				strings.Join(last.GetPermissionDenied().GetResources(), ","),
				"/tables/",
				"CREATE SEQUENCE must be denied via the bare-database fallback (empty write targets), not a per-target /tables/ check",
			)
		})
	})

	// 9. Codex P1 search_path-override bypass guard (SUP-222 / BYT-9698). Postgres
	//    honors QueryRequest.schema by running `SET search_path TO "<schema>"` before
	//    execution, so an UNQUALIFIED `INSERT INTO t_sbx` writes to <request_schema>.t_sbx.
	//    The grant is table-scoped to public.t_sbx. The user sends Schema="other_schema"
	//    + an unqualified INSERT: Postgres would write other_schema.t_sbx (NOT granted).
	//    The fix resolves the write target against the request schema (other_schema), so
	//    the public-scoped grant doesn't cover it → DENIED on other_schema/tables/t_sbx.
	//    Before the fix, resolution ignored the request schema and used public, wrongly
	//    ALLOWING the write to other_schema.t_sbx — the fail-open authorization bypass.
	//    The target schema/table need not exist: authorization precedes execution.
	t.Run("RequestSchemaOverrideNotBypassed", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		// Table-scoped DML grant on public.t_sbx only.
		cond := fmt.Sprintf(
			`resource.database == "%s" && resource.schema_name == "public" && resource.table_name in ["t_sbx"] && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		// Ensure public.t_sbx exists (the granted target) via the change-database flow.
		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE IF NOT EXISTS public.t_sbx (id int);`)},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		// Unqualified INSERT + Schema="other_schema": Postgres SET search_path would route
		// the write to other_schema.t_sbx, which the public-scoped grant does not cover.
		ctl.authInterceptor.token = token
		resp, qErr := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: "INSERT INTO t_sbx VALUES (1)",
			Schema:    new("other_schema"),
			Limit:     1000,
		}))
		ctl.authInterceptor.token = ownerToken

		var msg *v1pb.QueryResponse
		if resp != nil {
			msg = resp.Msg
		}
		// DENIED, naming other_schema (the real write target), NOT public.
		assertDeniedOn(t, msg, qErr, "schemas/other_schema/tables/t_sbx")
		last := msg.Results[len(msg.Results)-1]
		ra.NotContains(
			strings.Join(last.GetPermissionDenied().GetResources(), ","),
			"schemas/public/tables/t_sbx",
			"the request schema (other_schema), not public, must be the resolved write target",
		)
	})

	// 10. Codex P1 (round 2): an EARLIER statement in the same batch changes search_path.
	//     `SET search_path = other_schema; INSERT INTO t_sbx2` — SET is a safe Select (passes
	//     with the read grant), and Postgres runs both on the same session, so the unqualified
	//     INSERT writes other_schema.t_sbx2, which the public-scoped grant does not cover.
	//     Because a batch can redirect the schema mid-stream in forms the resolver can't
	//     reliably track, multi-statement DML/DDL falls back to the database-level check →
	//     DENIED at the bare database, so the redirected write is never authorized.
	//     SUP-222 / BYT-9698.
	t.Run("EarlierSetSearchPathNotBypassed", func(t *testing.T) {
		ra := require.New(t)
		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(
			`resource.database == "%s" && resource.schema_name == "public" && resource.table_name in ["t_sbx2"] && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE IF NOT EXISTS public.t_sbx2 (id int);`)},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		// An earlier SET search_path would redirect the later unqualified INSERT to
		// other_schema; the multi-statement batch must not authorize that write.
		ctl.authInterceptor.token = token
		resp, qErr := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: "SET search_path = other_schema; INSERT INTO t_sbx2 VALUES (1)",
			Limit:     1000,
		}))
		ctl.authInterceptor.token = ownerToken

		var msg *v1pb.QueryResponse
		if resp != nil {
			msg = resp.Msg
		}
		// Denied via the database-level fallback (multi-statement) — the bare database, not a
		// per-target /tables/ resource. The public-scoped grant cannot satisfy it.
		assertDeniedOn(t, msg, qErr, dbFullName)
		last := msg.Results[len(msg.Results)-1]
		ra.NotContains(
			strings.Join(last.GetPermissionDenied().GetResources(), ","),
			"/tables/",
			"a SET search_path batch must deny via the database-level fallback, never authorizing the redirected write",
		)
	})

	// 11. Sentinel/omit: for an UNqualified write with no request schema, the schema Postgres
	//     will resolve (connection user's $user/public) can't be known here, so the write
	//     target resolves to an internal sentinel and resource.schema_name is OMITTED
	//     (key-absent). Effects: schema-scoped grants fail closed (must qualify); table-only
	//     grants still match; and a negated schema condition must NOT over-allow (the
	//     empty-string trap). SUP-222 / BYT-9698.
	t.Run("UnqualifiedSchemaUnknown", func(t *testing.T) {
		ra := require.New(t)
		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE IF NOT EXISTS public.t_uq (id int);`)},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		// (a) schema-scoped grant + UNqualified write → denied; schema can't be confirmed, so
		//     the schema_name attribute is omitted and the `== "public"` clause errors. The
		//     denial names the table with NO /schemas/ segment.
		email, token := newLimitedUser(t)
		schemaCond := fmt.Sprintf(`resource.database == "%s" && resource.schema_name == "public" && resource.table_name in ["t_uq"] && resource.environment_id in ["%s"]`, dbFullName, envID)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, schemaCond))
		resp, qErr := runAs(token, "INSERT INTO t_uq VALUES (1)")
		assertDeniedOn(t, resp, qErr, "/tables/t_uq")
		ra.NotContains(strings.Join(resp.Results[len(resp.Results)-1].GetPermissionDenied().GetResources(), ","),
			"/schemas/", "unknown schema must omit the /schemas/ segment")

		// (b) Same grant, but a QUALIFIED write → schema is explicit (public) → allowed + executed.
		before := countRows(t, "public.t_uq")
		resp, qErr = runAs(token, "INSERT INTO public.t_uq VALUES (1)")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_uq"), "qualified write to the granted schema executes")

		// (c) TABLE-only grant (no schema clause) + UNqualified write → allowed (schema-agnostic).
		email2, token2 := newLimitedUser(t)
		tableCond := fmt.Sprintf(`resource.database == "%s" && resource.table_name in ["t_uq"] && resource.environment_id in ["%s"]`, dbFullName, envID)
		setProjectBindings(t, readBinding(email2), scopedBinding("roles/repro-write", email2, tableCond))
		before = countRows(t, "public.t_uq")
		resp, qErr = runAs(token2, "INSERT INTO t_uq VALUES (1)")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_uq"), "table-only grant allows the unqualified write")

		// (d) THE EMPTY-STRING TRAP: a NEGATED schema condition + UNqualified write must DENY.
		//     If schema_name were set to "" rather than omitted, `schema_name != "secret"`
		//     would evaluate true → over-allow. Omission (key-absent) makes CEL error → deny.
		email3, token3 := newLimitedUser(t)
		negCond := fmt.Sprintf(`resource.database == "%s" && resource.table_name in ["t_uq"] && resource.schema_name != "secret" && resource.environment_id in ["%s"]`, dbFullName, envID)
		setProjectBindings(t, readBinding(email3), scopedBinding("roles/repro-write", email3, negCond))
		resp, qErr = runAs(token3, "INSERT INTO t_uq VALUES (1)")
		assertDeniedOn(t, resp, qErr, "/tables/t_uq")
	})

	// 12. Case-folding: an unquoted uppercase `PUBLIC.t` folds to `public` and matches a
	//     public-scoped grant (the resolved schema is normalized, not the literal text).
	t.Run("CaseFoldedQualifiedSchema", func(t *testing.T) {
		ra := require.New(t)
		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(`CREATE TABLE IF NOT EXISTS public.t_cf (id int);`)},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(`resource.database == "%s" && resource.schema_name == "public" && resource.table_name in ["t_cf"] && resource.environment_id in ["%s"]`, dbFullName, envID)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))
		before := countRows(t, "public.t_cf")
		resp, qErr := runAs(token, "INSERT INTO PUBLIC.t_cf VALUES (1)")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_cf"), "case-folded PUBLIC matches the public-scoped grant")
	})

	// 13. TRUNCATE is table-scopable DDL (SUP-222 / BYT-9698, Codex follow-up). TRUNCATE
	//     classifies as DDL and names an explicit base-table target, so the changed-resource
	//     extractor resolves that target and a table-scoped bb.sql.ddl grant authorizes a
	//     TRUNCATE on the granted table and denies it per-target on any other — it does NOT
	//     fall back to the database-level check the way non-table DDL (CREATE SEQUENCE) does.
	t.Run("TruncateScopedDDL", func(t *testing.T) {
		ra := require.New(t)
		// Dedicated, seeded table so the allow case can prove execution (rows → 0).
		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte("CREATE TABLE IF NOT EXISTS public.t_trunc (id int);\nINSERT INTO public.t_trunc VALUES (1), (2), (3);")},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		email, token := newLimitedUser(t)
		ddlCond := fmt.Sprintf(
			`resource.database == "%s" && resource.table_name in ["t_trunc"] && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-ddl", email, ddlCond))

		// TRUNCATE on the granted table is authorized AND executes (all rows removed).
		ra.Greater(countRows(t, "public.t_trunc"), int64(0), "precondition: t_trunc must be seeded")
		resp, qErr := runAs(token, "TRUNCATE TABLE public.t_trunc;")
		assertAllowed(t, resp, qErr)
		ra.Equal(int64(0), countRows(t, "public.t_trunc"), "TRUNCATE should have removed all rows")

		// TRUNCATE on a non-granted table is denied per-target (names the table, not the bare db).
		resp, qErr = runAs(token, "TRUNCATE TABLE public.t_other;")
		assertDeniedOn(t, resp, qErr, "/tables/t_other")
	})

	// 14. resource.environment_id is resolved per TARGET database, not copied from the request
	//     database (SUP-222 / BYT-9698, Codex follow-up). A cross-database write names a database
	//     that may sit in a different environment; an environment-scoped grant must be evaluated
	//     against the TARGET's environment, or it could authorize a write into another
	//     environment. Postgres can't execute cross-database writes, but authorization precedes
	//     execution and the resolver keys on the statement's catalog name, so the access decision
	//     is observable: a second database on the same instance is placed in the `test`
	//     environment, and a grant scoped to the request database's `prod` environment must NOT
	//     authorize a write whose catalog names that test-environment database.
	t.Run("CrossDatabaseEnvironmentResolvedPerTarget", func(t *testing.T) {
		ra := require.New(t)
		testEnv, err := ctl.getEnvironment(ctx, "test")
		ra.NoError(err)
		// A second database on the SAME instance, in a DIFFERENT (test) environment.
		const otherDBName = "sup222edge_other_env"
		ra.NoError(ctl.createDatabase(ctx, ctl.project, instance, testEnv, otherDBName, "postgres"))

		// Seed the granted table in the request (prod) database for the positive control.
		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte("CREATE TABLE IF NOT EXISTS public.t_xenv (id int);")},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		// Grant: environment-scoped to the request database's environment (prod) + table t_xenv,
		// with NO database clause — so only the environment distinguishes the two targets.
		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(
			`resource.environment_id in ["%s"] && resource.table_name in ["t_xenv"]`,
			envID, // the request database's environment (prod)
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		// Positive control: a write to the request (prod) database matches the prod-scoped grant.
		before := countRows(t, "public.t_xenv")
		resp, qErr := runAs(token, "INSERT INTO public.t_xenv VALUES (1);")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_xenv"), "same-environment write executes")

		// The fix: a cross-database write whose catalog names the TEST-environment database is
		// DENIED — its environment is resolved from the target, not copied from the request. The
		// denial names the target database. (Pre-fix the request's prod environment was used, so
		// the prod-scoped grant wrongly authorized this write.)
		resp, qErr = runAs(token, fmt.Sprintf("INSERT INTO %s.public.t_xenv VALUES (1);", otherDBName))
		assertDeniedOn(t, resp, qErr, fmt.Sprintf("databases/%s", otherDBName))
	})

	// 15. A QUALIFIED cross-database write in a MULTI-statement batch must be authorized against
	//     its TARGET database, not the request database (SUP-222 / BYT-9698, Codex follow-up).
	//     Multi-statement batches drop to a database-level check (an earlier statement can rebind
	//     the session schema/database, so per-table/schema scoping is unreliable), but that check
	//     must still key on each target database: a grant on the request database must not
	//     authorize a write whose catalog names another database — otherwise wrapping the write in
	//     a batch would bypass the per-target check the single-statement path applies. (Postgres
	//     can't execute cross-database writes, but authorization precedes execution and keys on the
	//     catalog name, so the decision is observable.)
	t.Run("MultiStatementCrossDatabaseNotBypassed", func(t *testing.T) {
		ra := require.New(t)
		const otherDBName = "sup222edge_mstmt_other"
		ra.NoError(ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, otherDBName, "postgres"))

		// Seed the granted table in the request database for the positive control.
		setupSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte("CREATE TABLE IF NOT EXISTS public.t_mxdb (id int);")},
		}))
		ra.NoError(err)
		ra.NoError(ctl.changeDatabase(ctx, ctl.project, database, setupSheetResp.Msg, false))

		// DATABASE-scoped DML grant on the REQUEST database only (no table/schema clause).
		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(
			`resource.database == "%s" && resource.environment_id in ["%s"]`,
			dbFullName, envID,
		)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		// Positive control: a SAME-database multi-statement batch is authorized at the database
		// level by the request-database grant, and executes.
		before := countRows(t, "public.t_mxdb")
		resp, qErr := runAs(token, "SELECT 1; INSERT INTO public.t_mxdb VALUES (1);")
		assertAllowed(t, resp, qErr)
		ra.Equal(before+1, countRows(t, "public.t_mxdb"), "same-database multi-statement write executes")

		// The fix: the same request-database grant must NOT authorize a multi-statement batch
		// whose qualified write targets ANOTHER database — the database-level fallback checks the
		// TARGET database, not the request database. Denied on the other database.
		resp, qErr = runAs(token, fmt.Sprintf("SELECT 1; INSERT INTO %s.public.t_mxdb VALUES (1);", otherDBName))
		assertDeniedOn(t, resp, qErr, fmt.Sprintf("databases/%s", otherDBName))
	})

	// 16. MERGE resolves its write target (the INTO table) and is authorized per-target like
	//     INSERT/UPDATE/DELETE — not via the database-level fallback (SUP-222 / BYT-9698, Codex
	//     P1: an unmodeled MERGE would resolve to zero targets and authorize the qualified write
	//     against the request database). A table-scoped DML grant on the MERGE target authorizes
	//     it; MERGE into a non-granted table is denied per-target. (MERGE's USING read source is
	//     ungated — the documented read-source limitation.)
	t.Run("MergeAuthorizedPerTarget", func(t *testing.T) {
		email, token := newLimitedUser(t)
		setProjectBindings(t, readBinding(email), tableScopedDML(email, "t_dst"))

		mergeInto := func(target string) string {
			return fmt.Sprintf("MERGE INTO public.%s d USING public.t_src s ON d.id = s.id "+
				"WHEN NOT MATCHED THEN INSERT (id) VALUES (s.id);", target)
		}
		// MERGE into the granted table is authorized per-target (not a bare-database denial).
		resp, qErr := runAs(token, mergeInto("t_dst"))
		assertAllowed(t, resp, qErr)
		// MERGE into a non-granted table is denied, naming that table.
		resp, qErr = runAs(token, mergeInto("t_other"))
		assertDeniedOn(t, resp, qErr, "/tables/t_other")
	})

	// 17. A write target in a DIFFERENT project than the SQL-Editor session is denied. The
	//     per-target check evaluates the REQUEST project's IAM policy, so a cross-project target
	//     must fail closed to preserve the project boundary — otherwise a binding in the session's
	//     project could authorize a write to a database owned by another project. SUP-222.
	t.Run("CrossProjectWriteDenied", func(t *testing.T) {
		ra := require.New(t)
		ctl.authInterceptor.token = ownerToken
		projectID := generateRandomString("sup222other")
		_, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
			Project:   &v1pb.Project{Name: fmt.Sprintf("projects/%s", projectID), Title: projectID, AllowSelfApproval: true},
			ProjectId: projectID,
		}))
		ra.NoError(err)
		otherProject, err := ctl.projectServiceClient.GetProject(ctx, connect.NewRequest(&v1pb.GetProjectRequest{Name: fmt.Sprintf("projects/%s", projectID)}))
		ra.NoError(err)
		// A database in the OTHER project, on the SAME instance.
		const otherProjDB = "sup222edge_other_proj"
		ra.NoError(ctl.createDatabase(ctx, otherProject.Msg, instance, nil /* environment */, otherProjDB, "postgres"))

		// An ENVIRONMENT-scoped DML grant in the session's project (no database clause), so it
		// WOULD match the cross-project target's environment — proving the denial comes from the
		// cross-project guard, not a non-matching condition.
		email, token := newLimitedUser(t)
		cond := fmt.Sprintf(`resource.environment_id in ["%s"]`, envID)
		setProjectBindings(t, readBinding(email), scopedBinding("roles/repro-write", email, cond))

		resp, qErr := runAs(token, fmt.Sprintf("INSERT INTO %s.public.t VALUES (1);", otherProjDB))
		assertDeniedOn(t, resp, qErr, fmt.Sprintf("databases/%s", otherProjDB))
	})
}
