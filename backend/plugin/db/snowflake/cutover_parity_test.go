package snowflake

import (
	"fmt"
	"strings"
	"testing"

	snowflakeanalysis "github.com/bytebase/omni/snowflake/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowflakeparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

// TestCutoverParityStatements verifies statement splitting, limit rewriting,
// explain rewriting, and query classification between the pre-cutover
// parser/base path and the omni snowflake/analysis functions that
// omni/snowflake/driver calls, and pins the deliberate cutover differences.
func TestCutoverParityStatements(t *testing.T) {
	const limit = 50
	for _, script := range []string{
		"SELECT 1; SELECT 2;",
		"WITH c AS (SELECT 1 AS id) SELECT id FROM c",
		"SHOW TABLES; DESCRIBE TABLE t;",
		"INSERT INTO t VALUES (1); UPDATE t SET a = 2; DELETE FROM t;",
		"CREATE TABLE t (id INT); DROP TABLE t;",
		"BEGIN\n  INSERT INTO t VALUES (1);\n  RETURN 1;\nEND;",
	} {
		t.Run(script, func(t *testing.T) {
			oldDecision, err := oldStatementDecisions(script, limit)
			require.NoError(t, err)
			newDecision, err := newStatementDecisions(script, limit, false)
			require.NoError(t, err)
			require.Equal(t, oldDecision, newDecision)
		})
	}
}

func TestCutoverParityKnownDifferences(t *testing.T) {
	const limit = 10

	t.Run("a fallback limit wrapper puts a newline before its closing parenthesis", func(t *testing.T) {
		// OFFSET without LIMIT makes RewriteResultLimit fall back to a subquery
		// wrapper; a trailing line comment inside the statement would swallow
		// the closing parenthesis without the newline.
		stmt := "SELECT 1 OFFSET 5 -- trailing comment"
		oldLimited := base.StatementWithResultLimit(storepb.Engine_SNOWFLAKE, stmt, limit, "")
		require.Equal(t, "SELECT * FROM (SELECT 1 OFFSET 5 -- trailing comment) LIMIT 10", oldLimited)

		_, err := snowflakeanalysis.RewriteResultLimit(stmt, limit)
		require.Error(t, err)
		newLimited := fmt.Sprintf("SELECT * FROM (%s\n) LIMIT %d", base.TrimStatement(stmt), limit)
		require.Equal(t, "SELECT * FROM (SELECT 1 OFFSET 5 -- trailing comment\n) LIMIT 10", newLimited)
	})

	t.Run("a rewritten EXPLAIN statement carries QueryPlan metadata", func(t *testing.T) {
		for _, format := range []v1pb.QueryOption_ExplainFormat{
			v1pb.QueryOption_TEXT,
			v1pb.QueryOption_JSON,
		} {
			explained, err := base.ExplainStatement(storepb.Engine_SNOWFLAKE, "SELECT 1", db.ExplainFormat(format))
			require.NoError(t, err)
			qCtx := db.QueryContext{Option: &v1pb.QueryOption{ExplainFormat: format}}
			omniExplained, err := snowflakeanalysis.ExplainStatement("SELECT 1", strings.EqualFold(explainFormat(qCtx), "JSON"))
			require.NoError(t, err)
			require.Equal(t, explained, omniExplained)

			// The old driver skipped DescribeExplain when queryContext.Explain
			// was true; omni/snowflake/driver runs DescribeExplain on the
			// rewritten statement so the result carries its plan format.
			planFormat, executed, ok := snowflakeanalysis.DescribeExplain(omniExplained)
			require.True(t, ok)
			require.False(t, executed)
			require.Equal(t, db.TypedExplainPlan(snowflakeparser.DescribeExplain(omniExplained)), db.TypedExplainPlan(planFormat, executed, ok))
		}
	})

	t.Run("a procedure call runs as a query only when masking is disabled", func(t *testing.T) {
		stmt := "CALL my_proc(1)"
		_, oldAllQuery, err := base.ValidateSQLForEditor(storepb.Engine_SNOWFLAKE, stmt)
		require.NoError(t, err)
		require.False(t, oldAllQuery)

		unmasked, err := newStatementDecisions(stmt, 0, false)
		require.NoError(t, err)
		require.Equal(t, []bool{true}, unmasked.returnsRows)

		masked, err := newStatementDecisions(stmt, 0, true)
		require.NoError(t, err)
		require.Equal(t, []bool{false}, masked.returnsRows)
	})

	t.Run("an unparseable statement is refused when masking is enabled", func(t *testing.T) {
		stmt := "SELECT $1 FROM @my_stage"
		_, oldAllQuery, err := base.ValidateSQLForEditor(storepb.Engine_SNOWFLAKE, stmt)
		require.Error(t, err)
		// The old driver logged the parse error and fell back to running the
		// statement as a query even when masking was enabled.
		_ = oldAllQuery

		unmasked, err := newStatementDecisions(stmt, 0, false)
		require.NoError(t, err)
		require.Equal(t, []bool{true}, unmasked.returnsRows)

		_, err = newStatementDecisions(stmt, 0, true)
		require.Error(t, err)
	})
}

type statementDecision struct {
	statements  []string
	returnsRows []bool
	limited     []string
}

func oldStatementDecisions(script string, limit int) (statementDecision, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_SNOWFLAKE, script)
	if err != nil {
		return statementDecision{}, err
	}
	var out statementDecision
	for _, singleSQL := range singleSQLs {
		stmt := singleSQL.Text
		out.statements = append(out.statements, stmt)
		if limit > 0 {
			out.limited = append(out.limited, base.StatementWithResultLimit(storepb.Engine_SNOWFLAKE, stmt, limit, ""))
		}
		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_SNOWFLAKE, stmt)
		if err != nil {
			allQuery = true
		}
		out.returnsRows = append(out.returnsRows, allQuery)
	}
	return out, nil
}

func newStatementDecisions(script string, limit int, maskingEnabled bool) (statementDecision, error) {
	var out statementDecision
	for _, seg := range snowflakeanalysis.SplitStatements(script) {
		if seg.Empty {
			continue
		}
		stmt := seg.Text
		out.statements = append(out.statements, stmt)
		if limit > 0 {
			if rewritten, err := snowflakeanalysis.RewriteResultLimit(stmt, limit); err == nil {
				out.limited = append(out.limited, rewritten)
			} else {
				out.limited = append(out.limited, fmt.Sprintf("SELECT * FROM (%s\n) LIMIT %d", base.TrimStatement(stmt), limit))
			}
		}

		_, returns, err := snowflakeanalysis.ValidateQuery(stmt)
		if err != nil {
			if maskingEnabled {
				return statementDecision{}, err
			}
			returns = true
		} else if fields := strings.Fields(stmt); len(fields) > 0 && strings.EqualFold(fields[0], "CALL") {
			returns = !maskingEnabled
		}
		out.returnsRows = append(out.returnsRows, returns)
	}
	return out, nil
}
