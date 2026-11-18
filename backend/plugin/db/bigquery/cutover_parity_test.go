package bigquery

import (
	"context"
	"errors"
	"strings"
	"testing"

	odriver "github.com/bytebase/omni/driver"
	googlesqlanalysis "github.com/bytebase/omni/googlesql/analysis"
	"github.com/bytebase/omni/googlesql/parser"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Registers the BigQuery limit function the pre-cutover driver called, which
	// the previous db/bigquery pulled in for its own call.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/bigquery"
)

func TestBigQueryConfig(t *testing.T) {
	for _, tc := range []struct {
		name             string
		dataSource       *storepb.DataSource
		connectionConfig db.ConnectionConfig
		want             odriver.Config
	}{
		{
			name: "the project is a parameter and the dataset is the database",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				IamExtension: &storepb.DataSource_GcpCredential{
					GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"service_account"}`},
				},
			},
			connectionConfig: db.ConnectionConfig{
				ConnectionContext: db.ConnectionContext{DatabaseName: "omni_record"},
			},
			want: odriver.Config{
				Database: "omni_record",
				Password: `{"type":"service_account"}`,
				Params:   map[string]string{"project": "example-project"},
			},
		},
		{
			// The dataset is the database the driver was opened on, which is what
			// the previous driver read; a data source's own database field is not a
			// fallback for it.
			name: "the data source's database field is not the dataset",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				Database:  "from_data_source",
			},
			want: odriver.Config{
				Params: map[string]string{"project": "example-project"},
			},
		},
		{
			name: "a resolved credential wins over the data source's",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				IamExtension: &storepb.DataSource_GcpCredential{
					GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"service_account"}`},
				},
			},
			connectionConfig: db.ConnectionConfig{Password: `{"type":"authorized_user"}`},
			want: odriver.Config{
				Password: `{"type":"authorized_user"}`,
				Params:   map[string]string{"project": "example-project"},
			},
		},
		{
			name: "the endpoint override is the host and port",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				Host:      "bigquery.p.googleapis.com",
				Port:      "443",
			},
			want: odriver.Config{
				Host:   "bigquery.p.googleapis.com",
				Port:   "443",
				Params: map[string]string{"project": "example-project"},
			},
		},
		{
			// The client library owns its transport and holds no dialer, so a
			// data source's TLS and SSH settings reach no field, which is what the
			// previous driver did with them.
			name: "an SSL data source sets no TLS and an SSH one no dialer",
			dataSource: &storepb.DataSource{
				ProjectId:            "example-project",
				UseSsl:               true,
				VerifyTlsCertificate: true,
				SshHost:              "bastion.example.com",
				SshPort:              "22",
				SshUser:              "user",
			},
			want: odriver.Config{
				Params: map[string]string{"project": "example-project"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.connectionConfig
			config.DataSource = tc.dataSource
			require.Equal(t, tc.want, bigQueryConfig(config))
		})
	}
}

// TestOpenRequiresAProject pins the one check the previous driver made before it
// built a client. BigQuery's driver states it too, so Open answers before any
// credential is resolved or network reached.
func TestOpenRequiresAProject(t *testing.T) {
	_, err := newDriver().Open(context.Background(), storepb.Engine_BIGQUERY, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Database: "omni_record"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "project")
}

// TestQueryResultConversion pins the shapes the API receives for the results the
// omni driver can answer without a project: a statement's error, a statement's
// affected rows, a capped read, and a typed cell. Which cell kinds the driver
// answers is omni's own test (bigquery/driver TestConvertValue); this pins the
// conversion every adapter shares.
func TestQueryResultConversion(t *testing.T) {
	t.Run("a statement's error keeps its text and adds no rows", func(t *testing.T) {
		result := util.OmniResultToQueryResult(&odriver.Result{
			Statement: "SELEKT 1",
			Err:       errors.New(`Syntax error: Unexpected identifier "SELEKT" at [1:1], invalidQuery`),
		}, 0)
		require.Equal(t, "SELEKT 1", result.Statement)
		require.Equal(t, `Syntax error: Unexpected identifier "SELEKT" at [1:1], invalidQuery`, result.Error)
		require.Empty(t, result.ColumnNames)
		require.Zero(t, result.RowsCount)
	})

	t.Run("a command reports one Affected Rows row", func(t *testing.T) {
		affected := int64(3)
		result := util.OmniResultToQueryResult(&odriver.Result{
			Statement:    "INSERT INTO t VALUES (1), (2), (3)",
			AffectedRows: &affected,
		}, 0)
		require.Equal(t, []string{"Affected Rows"}, result.ColumnNames)
		require.Equal(t, []string{"INT"}, result.ColumnTypeNames)
		require.Len(t, result.Rows, 1)
		require.Equal(t, affected, result.Rows[0].Values[0].GetInt64Value())
	})

	t.Run("a capped read reports the API's cap message", func(t *testing.T) {
		const maxResultBytes = 1 << 20
		result := util.OmniResultToQueryResult(&odriver.Result{
			Statement: "SELECT * FROM t",
			Columns:   []odriver.Column{{Name: "a", TypeName: "INT64"}},
			Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
			Truncated: true,
		}, maxResultBytes)
		require.Equal(t, util.FormatMaximumSQLResultSizeMessage(maxResultBytes), result.Error)
		require.Len(t, result.Rows, 1)
	})

	t.Run("the plan of a finished read keeps its JSON", func(t *testing.T) {
		result := util.OmniResultToQueryResult(&odriver.Result{
			Statement: "SELECT 1",
			Plan: &odriver.Plan{
				Format:   "JSON",
				Executed: true,
				Content:  `[{"id":"0"}]`,
			},
		}, 0)
		require.NotNil(t, result.QueryPlan)
		require.Equal(t, v1pb.QueryOption_JSON, result.QueryPlan.Format)
		require.True(t, result.QueryPlan.Executed)
		require.Equal(t, `[{"id":"0"}]`, result.QueryPlan.Content)
	})

	t.Run("a typed cell keeps its kind", func(t *testing.T) {
		result := util.OmniResultToQueryResult(&odriver.Result{
			Columns: []odriver.Column{{Name: "n", TypeName: "NUMERIC"}},
			Rows: [][]odriver.Value{
				{{Kind: odriver.String, String: "12.5"}},
			},
		}, 0)
		require.Equal(t, "12.5", result.Rows[0].Values[0].GetStringValue())
	})
}

// TestStatementParityKnownDifferences pins each case the two sides read
// differently, both answers included, so the difference stays the one the cutover
// decided rather than a later drift. The cases where both sides agree are not
// re-asserted here: those are bigquery/driver's own dialect tests (its statement,
// returns-rows and limit suites), and the adapter passes the script through, so a
// second copy of them in this package would test the dependency rather than the
// seam.
func TestStatementParityKnownDifferences(t *testing.T) {
	t.Run("a comment stays in the statement", func(t *testing.T) {
		script := "SELECT 1 -- trailing\n"
		old, err := oldStatementDecision(script)
		require.NoError(t, err)
		require.Equal(t, []string{"SELECT 1"}, old.statements)
		require.Equal(t, []string{"WITH result AS (SELECT 1) SELECT * FROM result LIMIT 10;"}, old.limited)

		got := newStatementDecision(script)
		require.Equal(t, []string{"SELECT 1 -- trailing"}, got.statements)
		// The wrapper starts its closing parenthesis on a new line, because a
		// line comment would otherwise swallow it.
		require.Equal(t, []string{"WITH result AS (SELECT 1 -- trailing\n) SELECT * FROM result LIMIT 10;"}, got.limited)
	})

	t.Run("a parenthesized query returns rows", func(t *testing.T) {
		script := "(SELECT 1)"
		old, err := oldStatementDecision(script)
		require.NoError(t, err)
		require.Equal(t, []bool{false}, old.returnsRows)

		got := newStatementDecision(script)
		require.Equal(t, []bool{true}, got.returnsRows)
		require.Equal(t, old.statements, got.statements)
		require.Equal(t, old.limited, got.limited)
	})

	t.Run("a CTE-prefixed INSERT is a command", func(t *testing.T) {
		script := "WITH x AS (SELECT 1) INSERT INTO t SELECT * FROM x"
		old, err := oldStatementDecision(script)
		require.NoError(t, err)
		require.Equal(t, []bool{true}, old.returnsRows)

		got := newStatementDecision(script)
		require.Equal(t, []bool{false}, got.returnsRows)
		require.Equal(t, old.statements, got.statements)
	})

	t.Run("a pipe query the analysis cannot read is a command", func(t *testing.T) {
		script := "SELECT 1 |> WHERE TRUE"
		old, err := oldStatementDecision(script)
		require.NoError(t, err)
		require.Equal(t, []bool{true}, old.returnsRows)

		got := newStatementDecision(script)
		require.Equal(t, []bool{false}, got.returnsRows)
		// A statement the analysis cannot classify runs as written, so the CTE
		// wrap a classified query gets is not applied; the job's own answer keeps
		// the rows the service returned.
		require.Equal(t, []string{script}, got.statements)
	})

	t.Run("an unclosed literal is the statement's error", func(t *testing.T) {
		script := "SELECT 1;\nSELECT 'unclosed"
		_, err := oldStatementDecision(script)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unclosed literal")

		// The splitter never fails, so both statements reach the service and the
		// second answers its own error.
		got := newStatementDecision(script)
		require.Equal(t, []string{"SELECT 1", "SELECT 'unclosed"}, got.statements)
	})
}

// statementLimit is the row cap both sides rewrite with.
const statementLimit = 10

// statementDecision is one script's statement-level decisions: the statements the
// driver sends, whether each is expected to return rows, and each one's LIMIT
// rewrite.
type statementDecision struct {
	statements  []string
	returnsRows []bool
	limited     []string
}

func oldStatementDecision(script string) (statementDecision, error) {
	statements, err := util.SanitizeSQL(script)
	if err != nil {
		return statementDecision{}, err
	}
	var decision statementDecision
	for _, statement := range statements {
		decision.statements = append(decision.statements, statement)
		decision.returnsRows = append(decision.returnsRows, util.IsSelect(statement))
		decision.limited = append(decision.limited, base.StatementWithResultLimit(storepb.Engine_BIGQUERY, statement, statementLimit, ""))
	}
	return decision, nil
}

func newStatementDecision(script string) statementDecision {
	var decision statementDecision
	for _, segment := range parser.SplitFlat(script) {
		statement := strings.TrimSpace(segment.Text)
		if statement == "" {
			continue
		}
		decision.statements = append(decision.statements, statement)
		switch googlesqlanalysis.ClassifySQL(statement, googlesqlanalysis.DialectBigQuery) {
		case googlesqlanalysis.Select, googlesqlanalysis.SelectInfoSchema:
			decision.returnsRows = append(decision.returnsRows, true)
		default:
			decision.returnsRows = append(decision.returnsRows, false)
		}
		decision.limited = append(decision.limited, googlesqlanalysis.StatementWithResultLimit(statement, statementLimit, googlesqlanalysis.DialectBigQuery))
	}
	return decision
}
