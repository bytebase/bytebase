package spanner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	spannerdb "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	odriver "github.com/bytebase/omni/driver"
	spannerdriver "github.com/bytebase/omni/spanner/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

const (
	testProject      = "default"
	testInstance     = "default"
	testDatabase     = "adapter"
	testPostgresDB   = "adapterpg"
	testDDLDatabase  = "adapterexec"
	testEmulatorFlag = "emulator"
)

var testDDL = []string{
	`CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
) PRIMARY KEY (SingerId)`,
	`CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE`,
	`CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)`,
}

var testPostgresDDL = []string{
	`CREATE TABLE singers (
  singer_id bigint NOT NULL PRIMARY KEY,
  first_name text
)`,
}

// emulatorAddress returns the pinned emulator's address and skips without one.
// CI provides no Spanner, so every live test needs SPANNER_EMULATOR_HOST.
func emulatorAddress(t *testing.T) string {
	t.Helper()
	address := os.Getenv("SPANNER_EMULATOR_HOST")
	if address == "" {
		t.Skip("set SPANNER_EMULATOR_HOST to run against the pinned emulator")
	}
	return address
}

// emulatorOptions is how the fixture reaches the emulator: its address, no
// credential and no TLS.
func emulatorOptions(address string) []option.ClientOption {
	return []option.ClientOption{
		option.WithEndpoint(address),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}
}

// createDatabase returns database's resource name, creating the instance and
// the database with ddl as its statements. postgres asks for a PostgreSQL-dialect
// database, whose creation statement is written in that dialect.
func createDatabase(t *testing.T, address, database string, ddl []string, postgres bool) {
	t.Helper()
	ctx := context.Background()
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, emulatorOptions(address)...)
	require.NoError(t, err)
	defer instanceAdmin.Close()
	instanceOp, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", testProject),
		InstanceId: testInstance,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", testProject),
			DisplayName: testInstance,
			NodeCount:   1,
		},
	})
	if err != nil {
		require.Equal(t, codes.AlreadyExists, status.Code(err))
	} else {
		_, err = instanceOp.Wait(ctx)
		require.NoError(t, err)
	}

	parent := fmt.Sprintf("projects/%s/instances/%s", testProject, testInstance)
	request := &databasepb.CreateDatabaseRequest{
		Parent:          parent,
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", database),
		ExtraStatements: ddl,
	}
	if postgres {
		request.CreateStatement = fmt.Sprintf(`CREATE DATABASE "%s"`, database)
		request.DatabaseDialect = databasepb.DatabaseDialect_POSTGRESQL
	}
	admin, err := spannerdb.NewDatabaseAdminClient(ctx, emulatorOptions(address)...)
	require.NoError(t, err)
	defer admin.Close()
	databaseOp, err := admin.CreateDatabase(ctx, request)
	if err != nil {
		require.Equal(t, codes.AlreadyExists, status.Code(err), "create database %s: %v", database, err)
		return
	}
	_, err = databaseOp.Wait(ctx)
	require.NoError(t, err)
}

// openAdapter opens the adapter on database through the emulator's Params, not
// through a credential file or the client library's environment.
func openAdapter(t *testing.T, address, database string) *Driver {
	t.Helper()
	return openAdapterAt(t, address, testProject, testInstance, database)
}

func openAdapterAt(t *testing.T, address, project, instance, database string) *Driver {
	t.Helper()
	host, port, err := net.SplitHostPort(address)
	require.NoError(t, err)
	opened, err := db.Open(context.Background(), storepb.Engine_SPANNER, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			ProjectId:                 project,
			InstanceId:                instance,
			Host:                      host,
			Port:                      port,
			ExtraConnectionParameters: map[string]string{testEmulatorFlag: "1"},
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(context.Background()) })
	driver, ok := opened.(*Driver)
	require.True(t, ok)
	return driver
}

func TestAdapterAgainstTheEmulator(t *testing.T) {
	address := emulatorAddress(t)
	createDatabase(t, address, testDatabase, testDDL, false)
	driver := openAdapter(t, address, testDatabase)
	ctx := context.Background()

	require.NoError(t, driver.Ping(ctx))

	// The instance lists GoogleSQL databases only, which is the filter the old
	// driver applied because SPANNER's split and query span are GoogleSQL-only.
	createDatabase(t, address, testPostgresDB, testPostgresDDL, true)
	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	var names []string
	for _, database := range instance.Databases {
		names = append(names, database.Name)
	}
	require.Contains(t, names, testDatabase)
	require.NotContains(t, names, testPostgresDB)

	// The GoogleSQL database's schema reads the tables, the index and the
	// interleave's primary key.
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, testDatabase, metadata.Name)
	require.Len(t, metadata.Schemas, 1)
	var tables []string
	for _, table := range metadata.Schemas[0].Tables {
		tables = append(tables, table.Name)
	}
	require.ElementsMatch(t, []string{"Singers", "Albums"}, tables)

	// Dump is the server's DDL, statement by statement, and the synced metadata
	// does not narrow it: the old driver ignored the argument too.
	var dump bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &dump, metadata))
	require.Contains(t, dump.String(), "CREATE TABLE Singers (")
	require.Contains(t, dump.String(), "CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle);")
	var dumpWithoutMetadata bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &dumpWithoutMetadata, nil))
	require.Equal(t, dump.String(), dumpWithoutMetadata.String())

	// A PostgreSQL-dialect database is not reachable through the product while
	// the filter stays, but omni reads its own catalog if a caller opens it
	// directly. This is the capability the filter withholds, not a product path.
	postgres := openAdapter(t, address, testPostgresDB)
	postgresMetadata, err := postgres.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, postgresMetadata.Schemas, 1)
	require.Len(t, postgresMetadata.Schemas[0].Tables, 1)
	require.Equal(t, "singers", postgresMetadata.Schemas[0].Tables[0].Name)

	// A query's LIMIT is lowered into the statement the server runs.
	results, err := driver.QueryConn(ctx, `SELECT x FROM UNNEST([3, 1, 2]) AS x ORDER BY x`, db.QueryContext{
		Limit:                2,
		MaximumSQLResultSize: 100 * 1024 * 1024,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Equal(t, `SELECT x FROM UNNEST([3, 1, 2]) AS x ORDER BY x LIMIT 2`, results[0].Statement)
	var values []int64
	for _, row := range results[0].Rows {
		values = append(values, row.Values[0].GetInt64Value())
	}
	require.Equal(t, []int64{1, 2}, values)

	// A statement the server refuses rides on its result, and ends the script.
	results, err = driver.QueryConn(ctx, "SELECT * FROM NoSuchTable", db.QueryContext{
		MaximumSQLResultSize: 100 * 1024 * 1024,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotEmpty(t, results[0].Error)
}

func TestExecuteAgainstTheEmulator(t *testing.T) {
	address := emulatorAddress(t)
	createDatabase(t, address, testDDLDatabase, nil, false)
	driver := openAdapter(t, address, testDDLDatabase)
	ctx := context.Background()

	// A table of this run's own, so a rerun against the same emulator starts
	// empty.
	table := fmt.Sprintf("T%d", time.Now().UnixNano())
	_, err := driver.Execute(ctx, fmt.Sprintf("CREATE TABLE %s (Id INT64 NOT NULL, N INT64) PRIMARY KEY (Id)", table), db.ExecuteOptions{})
	require.NoError(t, err)
	affected, err := driver.Execute(ctx, fmt.Sprintf("INSERT OR UPDATE INTO %[1]s (Id, N) VALUES (1, 10); UPDATE %[1]s SET N = N + 1 WHERE Id = 1", table), db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)

	// OFF runs each statement as a partitioned update of its own.
	affected, err = driver.Execute(ctx, fmt.Sprintf("INSERT OR UPDATE INTO %s (Id, N) VALUES (2, 20)", table), db.ExecuteOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}

func TestSyncDBSchemaReportsAMissingDatabase(t *testing.T) {
	address := emulatorAddress(t)
	driver := openAdapter(t, address, "no_such_database")
	_, err := driver.SyncDBSchema(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), `database "no_such_database" not found`)
}

func TestToRowValue(t *testing.T) {
	tests := []struct {
		name  string
		value odriver.Value
		want  string
	}{
		{name: "null", value: odriver.Value{Kind: odriver.Null}, want: "null"},
		{name: "bool", value: odriver.Value{Kind: odriver.Bool, Bool: true}, want: "bool"},
		{name: "bytes", value: odriver.Value{Kind: odriver.Bytes, Bytes: []byte("hi")}, want: "bytes"},
		{name: "double", value: odriver.Value{Kind: odriver.Float64, Float64: 1.5}, want: "double"},
		{name: "int64", value: odriver.Value{Kind: odriver.Int64, Int64: 42}, want: "int64"},
		{name: "text", value: odriver.Value{Kind: odriver.String, String: "x"}, want: "string"},
		{name: "date is text", value: odriver.Value{Kind: odriver.String, String: "2024-01-15"}, want: "string"},
		{
			name:  "timestamp",
			value: odriver.Value{Kind: odriver.Timestamp, Time: time.Date(2024, 1, 15, 10, 30, 45, 123456000, time.UTC), Precision: 6},
			want:  "timestamp",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := toRowValue(test.value)
			require.NotNil(t, got)
			switch test.want {
			case "null":
				require.Equal(t, util.NullRowValue, got)
			case "bool":
				require.True(t, got.GetBoolValue())
			case "bytes":
				require.Equal(t, []byte("hi"), got.GetBytesValue())
			case "double":
				require.Equal(t, 1.5, got.GetDoubleValue())
			case "int64":
				require.Equal(t, int64(42), got.GetInt64Value())
			case "string":
				require.Equal(t, test.value.String, got.GetStringValue())
			case "timestamp":
				require.Equal(t, timestamppb.New(test.value.Time), got.GetTimestampValue().GetGoogleTimestamp())
				require.Equal(t, int32(6), got.GetTimestampValue().GetAccuracy())
			default:
				t.Fatalf("unknown want %q", test.want)
			}
		})
	}
}

// TestSpannerConfig checks what the adapter resolves into spanner/driver's
// Config: the resource name's two parts, the credential document, the caller's
// extra parameters and TLS.
func TestSpannerConfig(t *testing.T) {
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			ProjectId:  "p",
			InstanceId: "i",
			Host:       "10.0.0.1",
			Port:       "443",
			IamExtension: &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"authorized_user"}`},
			},
			ExtraConnectionParameters: map[string]string{"unknown": "x"},
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "d"},
	}
	got, err := spannerConfig(config)
	require.NoError(t, err)
	require.Equal(t, "p", got.Params[spannerdriver.ParamProject])
	require.Equal(t, "i", got.Params[spannerdriver.ParamInstance])
	require.Equal(t, "d", got.Database)
	require.Equal(t, "10.0.0.1", got.Host)
	require.Equal(t, "443", got.Port)
	require.Equal(t, `{"type":"authorized_user"}`, got.Password, "the data source's credential is the driver's password")
	require.Equal(t, "x", got.Params["unknown"])
	require.Nil(t, got.TLS, "TLS stays the client's unless the caller asks for it")

	// The emulator takes no credential, and the Google client refuses both at
	// once, so naming the emulator drops it.
	config.DataSource.ExtraConnectionParameters["emulator"] = "1"
	got, err = spannerConfig(config)
	require.NoError(t, err)
	require.Empty(t, got.Password)
	delete(config.DataSource.ExtraConnectionParameters, "emulator")

	// A caller that resolved a credential itself passes it as the password.
	config.Password = `{"type":"service_account"}`
	got, err = spannerConfig(config)
	require.NoError(t, err)
	require.Equal(t, `{"type":"service_account"}`, got.Password)

	// An SSL data source asks for TLS, bound to the host the driver dials.
	config.DataSource.UseSsl = true
	config.DataSource.VerifyTlsCertificate = true
	got, err = spannerConfig(config)
	require.NoError(t, err)
	require.NotNil(t, got.TLS)
	tlsConfig, err := got.TLS("10.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "10.0.0.1", tlsConfig.ServerName)
}

// TestOpenReadsTheCredentialDocument proves the credential path is the one the
// driver takes: a document the caller resolved reaches the client options, and
// one that is not a document is refused there.
func TestOpenReadsTheCredentialDocument(t *testing.T) {
	address := emulatorAddress(t)
	host, port, err := net.SplitHostPort(address)
	require.NoError(t, err)

	open := func(credential string, emulator bool) (db.Driver, error) {
		parameters := map[string]string{}
		if emulator {
			parameters[testEmulatorFlag] = "1"
		}
		return db.Open(context.Background(), storepb.Engine_SPANNER, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				ProjectId:                 testProject,
				InstanceId:                testInstance,
				Host:                      host,
				Port:                      port,
				ExtraConnectionParameters: parameters,
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: testDatabase},
			Password:          credential,
		})
	}

	// An emulator needs no credential at all.
	opened, err := open("", true)
	require.NoError(t, err)
	require.NoError(t, opened.Ping(context.Background()))
	require.NoError(t, opened.Close(context.Background()))

	// A credential document the caller resolved is read as one. The emulator
	// takes none, and the Google client refuses both at once, so the adapter
	// drops it for the emulator rather than fail Open.
	opened, err = open(`{"type":"authorized_user","client_id":"c","client_secret":"s","refresh_token":"r"}`, true)
	require.NoError(t, err)
	require.NoError(t, opened.Close(context.Background()))

	// One that is not a document is refused where the client options are built,
	// before any call, whatever the endpoint is.
	_, err = open("not a document", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse GCP credential JSON")
}

// TestExplainDMLWithoutRunningIt needs a Spanner that plans queries, which the
// emulator does not do and Spanner Omni does. SPANNER_TEST_DATABASE names the
// database by resource name, so the project and the instance come from it.
func TestExplainDMLWithoutRunningIt(t *testing.T) {
	database := os.Getenv("SPANNER_TEST_DATABASE")
	if database == "" {
		t.Skip("set SPANNER_EMULATOR_HOST and SPANNER_TEST_DATABASE to run against a server that plans queries")
	}
	address := emulatorAddress(t)
	// projects/<project>/instances/<instance>/databases/<database>
	parts := strings.Split(database, "/")
	require.Len(t, parts, 6, database)
	project, instanceName, databaseName := parts[1], parts[3], parts[5]

	driver := openAdapterAt(t, address, project, instanceName, databaseName)
	ctx := context.Background()
	table := fmt.Sprintf("ExplainDML%d", time.Now().UnixNano())
	_, err := driver.Execute(ctx, fmt.Sprintf("CREATE TABLE %s (Id INT64 NOT NULL, Total INT64) PRIMARY KEY (Id)", table), db.ExecuteOptions{})
	require.NoError(t, err)
	defer func() {
		_, _ = driver.Execute(context.Background(), fmt.Sprintf("DROP TABLE %s", table), db.ExecuteOptions{})
	}()
	_, err = driver.Execute(ctx, fmt.Sprintf("INSERT INTO %s (Id, Total) VALUES (1, 10)", table), db.ExecuteOptions{})
	require.NoError(t, err)

	results, err := driver.QueryConn(ctx, fmt.Sprintf(
		"UPDATE %[1]s SET Total = Total + 1 WHERE Id = 1; SELECT Total FROM %[1]s", table,
	), db.QueryContext{
		Explain:              true,
		MaximumSQLResultSize: 100 * 1024 * 1024,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		require.Empty(t, result.Error, result.Statement)
		require.Contains(t, result.Rows[0].Values[0].GetStringValue(), `"planNodes"`)
	}

	// Analysing the DML wrote nothing.
	results, err = driver.QueryConn(ctx, fmt.Sprintf("SELECT Total FROM %s WHERE Id = 1", table), db.QueryContext{
		MaximumSQLResultSize: 100 * 1024 * 1024,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, int64(10), results[0].Rows[0].Values[0].GetInt64Value())
}

// TestQueryConnLimitsQueryStartingWithWith keeps the two shapes the LIMIT
// rewrite decides: it lowers the outermost limit of a WITH query, and it leaves
// a non-literal limit alone, where the read caps the rows instead.
func TestQueryConnLimitsQueryStartingWithWith(t *testing.T) {
	address := emulatorAddress(t)
	createDatabase(t, address, testDatabase, testDDL, false)
	driver := openAdapter(t, address, testDatabase)
	ctx := context.Background()
	for _, statement := range []string{
		"WITH n AS (SELECT x FROM UNNEST([3, 1, 2]) AS x) SELECT x FROM n ORDER BY x",
		// The rewrite keeps a non-literal LIMIT, so the driver caps the rows.
		"WITH n AS (SELECT x FROM UNNEST([3, 1, 2]) AS x) SELECT x FROM n ORDER BY x LIMIT CAST(3 AS INT64)",
	} {
		results, err := driver.QueryConn(ctx, statement, db.QueryContext{
			Limit:                2,
			MaximumSQLResultSize: 100 * 1024 * 1024,
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error, results[0].Statement)
		var values []int64
		for _, row := range results[0].Rows {
			values = append(values, row.Values[0].GetInt64Value())
		}
		require.Equal(t, []int64{1, 2}, values, results[0].Statement)
	}
}

// TestGetDatabaseFromResource keeps the resource-name parse the old driver used
// to name a database in SyncInstance.
func TestGetDatabaseFromResource(t *testing.T) {
	tests := []struct {
		name string
		want string
		ok   bool
	}{
		{name: "projects/p/instances/i/databases/d", want: "d", ok: true},
		{name: "projects/p/instances/i/databases/", ok: false},
		{name: "projects/p/instances/i", ok: false},
	}
	for _, test := range tests {
		got, err := getDatabaseFromResource(test.name)
		if test.ok {
			require.NoError(t, err)
			require.Equal(t, test.want, got)
			continue
		}
		require.Error(t, err)
	}
}

// TestToQueryResultMatchesTheRecordedOldResults holds the adapter's result
// conversion to the shapes the old driver produced. The old path is deleted, so
// these are its recorded expectations: a query's columns, rows and row count; a
// DML statement's Affected Rows result, including the one row it reports; a DDL
// statement's empty result; and a failed statement's error with no rows.
func toQueryResult(result *odriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	return toQueryResults([]*odriver.Result{result}, maxResultBytes, false)[0]
}

func TestToQueryResultMatchesTheRecordedOldResults(t *testing.T) {
	maxBytes := int64(1024 * 1024)
	latency := time.Millisecond

	query := toQueryResult(&odriver.Result{
		Statement: "SELECT 1 AS n",
		Columns:   []odriver.Column{{Name: "n", TypeName: "INT64"}},
		Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
		Latency:   latency,
	}, maxBytes)
	require.Equal(t, "SELECT 1 AS n", query.Statement)
	require.Equal(t, []string{"n"}, query.ColumnNames)
	require.Equal(t, []string{"INT64"}, query.ColumnTypeNames)
	require.Equal(t, int64(1), query.RowsCount)
	require.Equal(t, int64(1), query.Rows[0].Values[0].GetInt64Value())

	affected := toQueryResult(&odriver.Result{
		Statement:    "UPDATE Singers SET Rating = 5 WHERE Id = 1",
		AffectedRows: func() *int64 { value := int64(1); return &value }(),
		Latency:      latency,
	}, maxBytes)
	require.Equal(t, []string{"Affected Rows"}, affected.ColumnNames)
	require.Equal(t, []string{"INT"}, affected.ColumnTypeNames)
	require.Equal(t, int64(1), affected.RowsCount, "the old driver counted the Affected Rows row")
	require.Equal(t, int64(1), affected.Rows[0].Values[0].GetInt64Value())

	ddl := toQueryResult(&odriver.Result{Statement: "CREATE TABLE T (Id INT64) PRIMARY KEY (Id)", Latency: latency}, maxBytes)
	require.Empty(t, ddl.ColumnNames)
	require.Empty(t, ddl.Rows)
	require.Equal(t, int64(0), ddl.RowsCount)

	failed := toQueryResult(&odriver.Result{Statement: "SELECT * FROM Missing", Err: errors.New("Table not found"), Latency: latency}, maxBytes)
	require.Equal(t, "Table not found", failed.Error)
	require.Empty(t, failed.Rows)
	require.Equal(t, int64(0), failed.RowsCount)
}

// TestMaxResultBytesCutIsTheDriversNotThePowerOfTwoCheck records the one
// result-shape change the adapter cannot reproduce: the old driver measured
// proto.Size at power-of-two row counts, and the contract cuts at the row that
// crosses the byte cap. The message is the same; the kept rows can differ.
func TestMaxResultBytesCutIsTheDriversNotThePowerOfTwoCheck(t *testing.T) {
	truncated := toQueryResult(&odriver.Result{
		Statement: "SELECT Id FROM Singers",
		Columns:   []odriver.Column{{Name: "Id", TypeName: "INT64"}},
		Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
		Truncated: true,
	}, 300)
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(300), truncated.Error)
	require.Equal(t, int64(1), truncated.RowsCount)
}

// TestQueryConnRunsEachStatementTheWayTheOldPathDid holds the query path to the
// recorded old results: a DML statement answers the Affected Rows row the old
// driver built, a SELECT answers its columns and rows, and a DDL statement
// answers an empty result.
func TestQueryConnRunsEachStatementTheWayTheOldPathDid(t *testing.T) {
	address := emulatorAddress(t)
	// A database of this run's own, so the writes cannot change the fixture the
	// other live tests read.
	database := fmt.Sprintf("querypath%d", time.Now().UnixNano())
	createDatabase(t, address, database, testDDL, false)
	driver := openAdapter(t, address, database)
	ctx := context.Background()

	table := fmt.Sprintf("QueryPath%d", time.Now().UnixNano())
	_, err := driver.Execute(ctx, fmt.Sprintf("CREATE TABLE %s (Id INT64 NOT NULL, N INT64) PRIMARY KEY (Id)", table), db.ExecuteOptions{})
	require.NoError(t, err)
	_, err = driver.Execute(ctx, fmt.Sprintf("INSERT INTO %s (Id, N) VALUES (1, 10)", table), db.ExecuteOptions{})
	require.NoError(t, err)

	results, err := driver.QueryConn(ctx, fmt.Sprintf(
		"UPDATE %[1]s SET N = N + 1 WHERE Id = 1; SELECT N FROM %[1]s WHERE Id = 1; CREATE TABLE %[1]sEmpty (Id INT64 NOT NULL) PRIMARY KEY (Id)",
		table,
	), db.QueryContext{MaximumSQLResultSize: 100 * 1024 * 1024})
	require.NoError(t, err)
	require.Len(t, results, 3)

	require.Equal(t, fmt.Sprintf("UPDATE %s SET N = N + 1 WHERE Id = 1", table), results[0].Statement)
	require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
	require.Equal(t, []string{"INT"}, results[0].ColumnTypeNames)
	require.Equal(t, int64(1), results[0].RowsCount)
	require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())

	require.Equal(t, fmt.Sprintf("SELECT N FROM %s WHERE Id = 1", table), results[1].Statement)
	require.Equal(t, []string{"N"}, results[1].ColumnNames)
	require.Equal(t, []string{"INT64"}, results[1].ColumnTypeNames)
	require.Equal(t, int64(1), results[1].RowsCount)
	require.Equal(t, int64(11), results[1].Rows[0].Values[0].GetInt64Value())

	require.Equal(t, fmt.Sprintf("CREATE TABLE %sEmpty (Id INT64 NOT NULL) PRIMARY KEY (Id)", table), results[2].Statement)
	require.Empty(t, results[2].ColumnNames)
	require.Empty(t, results[2].Rows)
	require.Equal(t, int64(0), results[2].RowsCount)
}

// TestExecuteReportsTransactionControl keeps the old Execute log: a DML script
// reports BEGIN before its transaction and COMMIT after it, and nothing else.
func TestExecuteReportsTransactionControl(t *testing.T) {
	address := emulatorAddress(t)
	createDatabase(t, address, testDatabase, testDDL, false)
	driver := openAdapter(t, address, testDatabase)
	ctx := context.Background()

	var logs []storepb.TaskRunLog_TransactionControl_Type
	_, err := driver.Execute(ctx, "UPDATE Singers SET FirstName = FirstName WHERE SingerId = 1", db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			if control := log.GetTransactionControl(); control != nil {
				logs = append(logs, control.Type)
			}
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, []storepb.TaskRunLog_TransactionControl_Type{
		storepb.TaskRunLog_TransactionControl_BEGIN,
		storepb.TaskRunLog_TransactionControl_COMMIT,
	}, logs)
}
