package v1

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSQLIAMDatabaseResourceUsesCanonicalInstanceScope(t *testing.T) {
	t.Parallel()
	projectID := "project-a"
	projectInstance := &store.InstanceMessage{ResourceID: "instance-a", ProjectID: &projectID}
	workspaceInstance := &store.InstanceMessage{ResourceID: "instance-b"}
	database := &store.DatabaseMessage{DatabaseName: "app"}
	projectDatabaseName := formatDatabaseResourceName(projectInstance, &store.DatabaseMessage{
		InstanceID:   projectInstance.ResourceID,
		DatabaseName: database.DatabaseName,
	})

	require.Equal(t,
		"projects/project-a/instances/instance-a/databases/app",
		projectDatabaseName,
	)
	require.Equal(t,
		"instances/instance-b/databases/app",
		formatDatabaseResourceName(workspaceInstance, &store.DatabaseMessage{
			InstanceID:   workspaceInstance.ResourceID,
			DatabaseName: database.DatabaseName,
		}),
	)

	attributes := map[string]any{common.CELAttributeResourceDatabase: projectDatabaseName}
	allowed, err := evaluateQueryExportPolicyCondition(
		`resource.database == "projects/project-a/instances/instance-a/databases/app"`, attributes)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = evaluateQueryExportPolicyCondition(
		`resource.database == "instances/instance-a/databases/app"`, attributes)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t,
		"projects/project-a/instances/instance-a/databases/app/schemas/public/tables/users",
		formatWriteTargetResource(projectDatabaseName, "public", "users"),
	)
}

// TestExportRejectsRetiredRolloutNames pins the retired export-data routes:
// rollout/stage names must fail name parsing with InvalidArgument, not fall
// through as 500s.
func TestExportRejectsRetiredRolloutNames(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.UserContextKey, &store.UserMessage{Email: "u@example.com"})
	s := &SQLService{}
	for _, name := range []string{
		"projects/p/plans/1/rollout",
		"projects/p/plans/1/rollout/stages/dev",
	} {
		_, err := s.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{Name: name, Statement: "SELECT 1"}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "name %q", name)
	}
}

// TestSelectBestAccessGrantPrefersUnmask covers the Unmask-only ranking:
// Export plays no role (Export callers already filtered the pool via the
// `requireExport` CEL filter, and Query never reads `Payload.Export`).
// Preferring Unmask=true is what addresses PR #20491 bot review
// (#3349086819) — a user with an active unmask grant should never be
// masked by Query when an export-only grant shares the slice.
func TestSelectBestAccessGrantPrefersUnmask(t *testing.T) {
	grantOf := func(unmask, export bool) *store.AccessGrantMessage {
		return &store.AccessGrantMessage{
			Payload: &storepb.AccessGrantPayload{Unmask: unmask, Export: export},
		}
	}

	t.Run("returns nil for empty input", func(t *testing.T) {
		require.Nil(t, selectBestAccessGrant(nil))
		require.Nil(t, selectBestAccessGrant([]*store.AccessGrantMessage{}))
	})

	t.Run("skips nil-payload grants and returns nil when none remain", func(t *testing.T) {
		got := selectBestAccessGrant([]*store.AccessGrantMessage{
			{Payload: nil},
			{Payload: nil},
		})
		require.Nil(t, got)
	})

	t.Run("returns the only valid grant when others have nil payloads", func(t *testing.T) {
		valid := grantOf(true, false)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{
			{Payload: nil},
			valid,
			{Payload: nil},
		})
		require.Same(t, valid, got)
	})

	t.Run("unmask grant wins over export-only grant regardless of slice order", func(t *testing.T) {
		unmaskGrant := grantOf(true, false)
		exportOnly := grantOf(false, true)
		// Put exportOnly first to prove the win isn't due to position —
		// Unmask scoring is what selects unmaskGrant.
		got := selectBestAccessGrant([]*store.AccessGrantMessage{exportOnly, unmaskGrant})
		require.Same(t, unmaskGrant, got)
	})

	t.Run("among multiple unmask grants the first in slice order wins", func(t *testing.T) {
		first := grantOf(true, false)
		second := grantOf(true, true)
		// Both have Unmask=true → tied score (Export plays no role). First
		// in slice wins via strict ">". Equivalent for Query callers
		// since both grants yield the same SkipMasking=true.
		got := selectBestAccessGrant([]*store.AccessGrantMessage{first, second})
		require.Same(t, first, got)
	})

	t.Run("returns a no-unmask grant when none have unmask", func(t *testing.T) {
		// All-zero-score case: a no-unmask grant is still returned (it
		// confers ACL bypass for Query, even if it can't unmask).
		exportOnly := grantOf(false, true)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{exportOnly})
		require.Same(t, exportOnly, got)
	})
}

// TestBuildExportQueryContextPropagatesSkipMasking pins the bug fix from PR
// #20487: when an export is authorized by a JIT grant with unmask=true,
// SkipMasking must reach db.QueryContext so the driver doesn't mask rows at
// query time. The earlier code only consulted skipMasking around the
// post-execution MaskResults pass — by then drivers like postgres with
// query-time masking rewrites had already returned masked rows.
func TestBuildExportQueryContextPropagatesSkipMasking(t *testing.T) {
	restriction := &store.EffectiveQueryDataPolicy{
		MaximumResultRows: 1000,
		MaximumResultSize: 1 << 20,
	}

	t.Run("grant with unmask=true sets SkipMasking", func(t *testing.T) {
		qc := buildExportQueryContext(restriction, "alice@example.com", nil, "", true)
		require.True(t, qc.SkipMasking, "SkipMasking must propagate so driver-level masking is bypassed")
	})

	t.Run("grant with unmask=false keeps SkipMasking false", func(t *testing.T) {
		qc := buildExportQueryContext(restriction, "alice@example.com", nil, "", false)
		require.False(t, qc.SkipMasking, "SkipMasking must stay false so masking still applies")
	})
}

// TestBuildExportQueryContextPropagatesOtherFields guards against accidental
// drops of the surrounding fields if buildExportQueryContext is edited.
func TestBuildExportQueryContextPropagatesOtherFields(t *testing.T) {
	t.Parallel()
	schema := "public"
	restriction := &store.EffectiveQueryDataPolicy{
		MaximumResultRows:        500,
		MaximumResultSize:        2 << 20,
		MaxQueryTimeoutInSeconds: 30,
	}

	qc := buildExportQueryContext(restriction, "alice@example.com", &schema, "customers", false)

	require.Equal(t, 500, qc.Limit)
	require.Equal(t, "alice@example.com", qc.OperatorEmail)
	require.Equal(t, int64(2<<20), qc.MaximumSQLResultSize)
	require.Equal(t, "public", qc.Schema)
	require.Equal(t, "customers", qc.Container)
	require.NotNil(t, qc.Timeout)
	require.Equal(t, int64(30), qc.Timeout.Seconds)
}

// TestBuildExportQueryContextOmitsTimeoutWhenZero verifies that an unset
// MaxQueryTimeoutInSeconds doesn't leak a zero-Duration timeout into the
// query context (which the driver layer treats differently from "no timeout").
func TestBuildExportQueryContextOmitsTimeoutWhenZero(t *testing.T) {
	t.Parallel()
	restriction := &store.EffectiveQueryDataPolicy{
		MaximumResultRows: 100,
		MaximumResultSize: 1 << 20,
	}

	qc := buildExportQueryContext(restriction, "alice@example.com", nil, "", false)
	require.Nil(t, qc.Timeout)
	require.Equal(t, "", qc.Schema)
}

func TestResolveDataSourceIDUsesAdminForNonReadOnlyAutomaticQueryWhenAllowed(t *testing.T) {
	t.Parallel()
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{
				{Id: "admin", Type: storepb.DataSourceType_ADMIN},
				{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
			},
		},
	}

	got, err := resolveDataSourceID(context.Background(), instance, "", "INSERT INTO books VALUES (1, 'Bytebase');", true)
	require.NoError(t, err)
	require.Equal(t, "admin", got)
}

func TestResolveDataSourceIDKeepsReadOnlyForReadOnlyAutomaticQueryWhenAllowed(t *testing.T) {
	t.Parallel()
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{
				{Id: "admin", Type: storepb.DataSourceType_ADMIN},
				{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
			},
		},
	}

	got, err := resolveDataSourceID(context.Background(), instance, "", "SELECT * FROM books;", true)
	require.NoError(t, err)
	require.Equal(t, "readonly", got)
}

func TestResolveDataSourceIDKeepsReadOnlyForNonReadOnlyAutomaticQueryWhenAdminDisallowed(t *testing.T) {
	t.Parallel()
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{
				{Id: "admin", Type: storepb.DataSourceType_ADMIN},
				{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
			},
		},
	}

	got, err := resolveDataSourceID(context.Background(), instance, "", "INSERT INTO books VALUES (1, 'Bytebase');", false)
	require.NoError(t, err)
	require.Equal(t, "readonly", got)
}

func TestResolveDataSourceIDUsesAdminForDocumentEngineAutomaticWriteQueryWhenAllowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		engine    storepb.Engine
		statement string
	}{
		{
			name:      "MongoDB DML",
			engine:    storepb.Engine_MONGODB,
			statement: `db.users.insertOne({name: "Bytebase"})`,
		},
		{
			name:      "MongoDB DDL",
			engine:    storepb.Engine_MONGODB,
			statement: `db.createCollection("users")`,
		},
		{
			name:      "Elasticsearch DML",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "POST /users/_doc\n{\"name\":\"Bytebase\"}",
		},
		{
			name:      "Elasticsearch DDL",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "PUT /users",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			instance := &store.InstanceMessage{
				Metadata: &storepb.Instance{
					Engine: tc.engine,
					DataSources: []*storepb.DataSource{
						{Id: "admin", Type: storepb.DataSourceType_ADMIN},
						{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
					},
				},
			}

			got, err := resolveDataSourceID(context.Background(), instance, "", tc.statement, true)
			require.NoError(t, err)
			require.Equal(t, "admin", got)
		})
	}
}

func TestResolveDataSourceIDKeepsReadOnlyForDocumentEngineAutomaticReadQueryWhenAllowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		engine    storepb.Engine
		statement string
	}{
		{
			name:      "MongoDB read",
			engine:    storepb.Engine_MONGODB,
			statement: `db.users.find({})`,
		},
		{
			name:      "Elasticsearch read",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "GET /users/_search",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			instance := &store.InstanceMessage{
				Metadata: &storepb.Instance{
					Engine: tc.engine,
					DataSources: []*storepb.DataSource{
						{Id: "admin", Type: storepb.DataSourceType_ADMIN},
						{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
					},
				},
			}

			got, err := resolveDataSourceID(context.Background(), instance, "", tc.statement, true)
			require.NoError(t, err)
			require.Equal(t, "readonly", got)
		})
	}
}

// TestValidateExplainFormat pins the engine-to-format support table. A format an
// engine cannot produce has to be refused here, because the drivers below map
// anything that reaches them onto their own default rather than failing.
func TestValidateExplainFormat(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		engine  storepb.Engine
		format  v1pb.QueryOption_ExplainFormat
		wantErr bool
	}{
		{name: "unspecified is always the engine default", engine: storepb.Engine_MYSQL, format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED},
		{name: "postgres json", engine: storepb.Engine_POSTGRES, format: v1pb.QueryOption_JSON},
		{name: "postgres xml", engine: storepb.Engine_POSTGRES, format: v1pb.QueryOption_XML},
		{name: "postgres text", engine: storepb.Engine_POSTGRES, format: v1pb.QueryOption_TEXT},
		{name: "postgres yaml", engine: storepb.Engine_POSTGRES, format: v1pb.QueryOption_YAML},
		{name: "mssql yaml", engine: storepb.Engine_MSSQL, format: v1pb.QueryOption_YAML, wantErr: true},
		{name: "mssql xml", engine: storepb.Engine_MSSQL, format: v1pb.QueryOption_XML},
		{name: "mssql json", engine: storepb.Engine_MSSQL, format: v1pb.QueryOption_JSON, wantErr: true},
		{name: "spanner json", engine: storepb.Engine_SPANNER, format: v1pb.QueryOption_JSON},
		{name: "spanner has no text plan", engine: storepb.Engine_SPANNER, format: v1pb.QueryOption_TEXT, wantErr: true},
		{name: "mysql text", engine: storepb.Engine_MYSQL, format: v1pb.QueryOption_TEXT},
		{name: "mysql json is not implemented yet", engine: storepb.Engine_MYSQL, format: v1pb.QueryOption_JSON, wantErr: true},
		{name: "oracle xml is not implemented yet", engine: storepb.Engine_ORACLE, format: v1pb.QueryOption_XML, wantErr: true},
		{name: "redis explains nothing at all", engine: storepb.Engine_REDIS, format: v1pb.QueryOption_TEXT, wantErr: true},
		{name: "mongodb explains nothing at all", engine: storepb.Engine_MONGODB, format: v1pb.QueryOption_JSON, wantErr: true},
		// A driver that ignores the explain flag runs the statement instead, and
		// the explain path skips the read-only validation, so an unspecified
		// format must be refused here too rather than reaching the driver.
		{name: "cassandra refuses even an unspecified format", engine: storepb.Engine_CASSANDRA, format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, wantErr: true},
		{name: "cosmosdb refuses even an unspecified format", engine: storepb.Engine_COSMOSDB, format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, wantErr: true},
		{name: "databricks refuses even an unspecified format", engine: storepb.Engine_DATABRICKS, format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, wantErr: true},
		{name: "elasticsearch refuses even an unspecified format", engine: storepb.Engine_ELASTICSEARCH, format: v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateExplainFormat(tc.engine, tc.format)
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}

// explainPrefixEngines are the engines that plan a statement by prefixing EXPLAIN
// to it, and so run a different statement than the one the caller sent.
//
// The engine parsers are registered for the whole v1 test binary by blank imports
// elsewhere in the package, so each call below returns that engine's real verdict
// rather than a no-parser default.
var explainPrefixEngines = []storepb.Engine{
	storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_MARIADB,
	storepb.Engine_OCEANBASE, storepb.Engine_TIDB, storepb.Engine_REDSHIFT,
	storepb.Engine_COCKROACHDB, storepb.Engine_SNOWFLAKE, storepb.Engine_CLICKHOUSE,
	storepb.Engine_STARROCKS, storepb.Engine_DORIS, storepb.Engine_HIVE,
	storepb.Engine_TRINO,
}

// TestExplainGateRejectsSmuggledWrite locks the smuggle defense. An explain
// request carries the bare statement, so "ANALYZE DELETE FROM t" — not valid SQL
// on its own — would become EXPLAIN ANALYZE DELETE and execute the DELETE if
// anything simply prefixed it. Every prefix engine must refuse it, by parsing the
// statement it was handed or by classifying the planned form as non-read-only, so
// it never reaches the driver.
func TestExplainGateRejectsSmuggledWrite(t *testing.T) {
	t.Parallel()
	for _, engine := range explainPrefixEngines {
		t.Run(engine.String(), func(t *testing.T) {
			t.Parallel()
			instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: engine}}
			require.Error(t, validateExplainStatements(instance, "ANALYZE DELETE FROM t", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED),
				"EXPLAIN ANALYZE DELETE must not pass the read-only gate")
		})
	}
}

// TestExplainGatePlansAPlanOnce locks the rebuild. A statement that already asks
// for a plan is planned once: the request's EXPLAIN replaces the caller's rather
// than wrapping it, which a second EXPLAIN would turn into a syntax error.
func TestExplainGatePlansAPlanOnce(t *testing.T) {
	t.Parallel()
	for _, engine := range explainPrefixEngines {
		t.Run(engine.String(), func(t *testing.T) {
			t.Parallel()
			instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: engine}}
			require.NoError(t, validateExplainStatements(instance, "EXPLAIN SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED))
		})
	}
}

// TestExplainGateNeverAnalyzes locks what the rebuild is for: an EXPLAIN ANALYZE
// of a write executes the write, so no explain request may end up running one,
// whatever the caller wrote. Each engine either plans the write without the
// caller's ANALYZE or refuses the statement — a dialect with no EXPLAIN ANALYZE
// cannot parse it, and ClickHouse and Hive have no AST to rebuild it with, so
// theirs runs as written and their read-only gate refuses it.
func TestExplainGateNeverAnalyzes(t *testing.T) {
	t.Parallel()
	for _, engine := range explainPrefixEngines {
		t.Run(engine.String(), func(t *testing.T) {
			t.Parallel()
			planned, err := parserbase.ExplainStatement(engine, "EXPLAIN ANALYZE DELETE FROM t", parserbase.ExplainFormatDefault)
			if err != nil {
				instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: engine}}
				require.Error(t, validateExplainStatements(instance, "EXPLAIN ANALYZE DELETE FROM t", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED),
					"a statement the parser will not plan must not reach the driver")
				return
			}
			if strings.Contains(strings.ToUpper(planned), "ANALYZE") {
				// The statement runs as the caller wrote it, so the read-only gate
				// is what stands between it and the DELETE.
				require.Equal(t, "EXPLAIN ANALYZE DELETE FROM t", planned)
				instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: engine}}
				require.Error(t, validateExplainStatements(instance, "EXPLAIN ANALYZE DELETE FROM t", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED))
				return
			}
			require.Equal(t, "EXPLAIN DELETE FROM t", planned)
		})
	}
}

// TestExplainGateAllowsPlans confirms the gate does not over-reject legitimate
// plans. A plain EXPLAIN plans without executing, so a read or a write is allowed;
// and because the driver splits a multi-statement request and prefixes EXPLAIN to
// each statement (pg.go and siblings), the gate does the same with the same
// per-engine splitter — a batch of plain writes is planned one at a time. A write
// smuggled into a batch as EXPLAIN ANALYZE is still refused wherever it sits;
// TestExplainGateRejectsSmuggledWrite is the single-statement sweep over every
// engine.
func TestExplainGateAllowsPlans(t *testing.T) {
	t.Parallel()
	pg := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES}}
	// Plain single-statement plans, read or write, are allowed.
	for _, stmt := range []string{"SELECT 1", "DELETE FROM t"} {
		require.NoError(t, validateExplainStatements(pg, stmt, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED), stmt)
	}
	// Engines that split on ';' plan each statement in its own non-executing EXPLAIN,
	// so a batch of plain writes is allowed rather than over-rejected.
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_MYSQL} {
		instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: engine}}
		require.NoErrorf(t, validateExplainStatements(instance, "DELETE FROM hello; DELETE FROM hello", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED), "%s: batch of plain writes", engine)
	}
	// A write smuggled into a batch is refused wherever it sits — on engines that
	// split it (pg, mysql) and on one whose splitter keeps it whole and validates
	// the unwrapped tail (clickhouse).
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_CLICKHOUSE} {
		instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: engine}}
		require.Errorf(t, validateExplainStatements(instance, "SELECT 1; ANALYZE DELETE FROM t", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED), "%s: smuggle after a read", engine)
		require.Errorf(t, validateExplainStatements(instance, "ANALYZE DELETE FROM t; SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED), "%s: smuggle before a read", engine)
	}
}
