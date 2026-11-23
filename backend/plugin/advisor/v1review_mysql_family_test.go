package advisor

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytebase/omni/v1review"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestMySQLFamilyColumnCommentNilPayload(t *testing.T) {
	stmt := V1Stmt[int]{Node: 1, Text: "CREATE TABLE t (id INT);", BaseLine: 0}
	extract := func(Context) ([]V1Stmt[int], error) {
		return []V1Stmt[int]{stmt}, nil
	}
	byteOffsetLine := func(string, int) int32 { return 1 }

	for _, tidbQuirks := range []bool{false, true} {
		advisors := buildMySQLFamilyAdvisors(
			storepb.Engine_MYSQL,
			newV1EngineWithOptions(v1EngineOptions[int]{
				Extract:        extract,
				ByteOffsetLine: byteOffsetLine,
				PosMode:        V1PosContentStart,
				TiDBQuirks:     tidbQuirks,
			}),
			MySQLFamilyOptions[int]{Extract: extract, ByteOffsetLine: byteOffsetLine, TiDBQuirks: tidbQuirks},
			MySQLFamilyRuleSet[int]{
				ColumnComment: func(_ []int, required bool, maxLength int) []v1review.Finding {
					if tidbQuirks {
						require.False(t, required)
						require.Equal(t, -1, maxLength)
						return []v1review.Finding{{Code: v1review.ColumnCommentTooLong, Table: "t", Name: "id"}}
					}
					require.True(t, required)
					require.Equal(t, 0, maxLength)
					return []v1review.Finding{{Code: v1review.ColumnCommentMissing, Table: "t", Name: "id"}}
				},
			},
		)
		checkMissing := advisors[storepb.SQLReviewRule_COLUMN_COMMENT]
		_, err := checkMissing.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_COLUMN_COMMENT,
				Level: storepb.SQLReviewRule_WARNING,
			},
		})
		require.ErrorContains(t, err, "comment_convention_payload is required")

		emptyAdvisors := buildMySQLFamilyAdvisors(
			storepb.Engine_MYSQL,
			newV1EngineWithOptions(v1EngineOptions[int]{
				Extract:        extract,
				ByteOffsetLine: byteOffsetLine,
				PosMode:        V1PosContentStart,
				TiDBQuirks:     tidbQuirks,
			}),
			MySQLFamilyOptions[int]{Extract: extract, ByteOffsetLine: byteOffsetLine, TiDBQuirks: tidbQuirks},
			MySQLFamilyRuleSet[int]{
				ColumnComment: func([]int, bool, int) []v1review.Finding { return nil },
			},
		)
		advice, err := emptyAdvisors[storepb.SQLReviewRule_COLUMN_COMMENT].Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_COLUMN_COMMENT,
				Level: storepb.SQLReviewRule_WARNING,
			},
		})
		require.NoError(t, err)
		require.Empty(t, advice)
	}
}

func TestMySQLFamilyPrepareMemoizesStmtsAndNodes(t *testing.T) {
	const stmtCount = 100
	extractCalls := 0
	opt := MySQLFamilyOptions[int]{
		Extract: func(Context) ([]V1Stmt[int], error) {
			extractCalls++
			stmts := make([]V1Stmt[int], stmtCount)
			for i := range stmts {
				stmts[i] = V1Stmt[int]{Node: i, Text: "SELECT 1"}
			}
			return stmts, nil
		},
		ByteOffsetLine: func(string, int) int32 { return 1 },
	}
	advisors := buildMySQLFamilyAdvisors(
		storepb.Engine_MYSQL,
		newV1EngineWithOptions(v1EngineOptions[int]{
			Extract:        opt.Extract,
			ByteOffsetLine: opt.ByteOffsetLine,
			PosMode:        V1PosContentStart,
		}),
		opt,
		MySQLFamilyRuleSet[int]{
			StatementDisallowCommit: func(nodes []int) []v1review.Finding {
				if len(nodes) != 1 {
					t.Fatalf("expected 1 node, got %d", len(nodes))
				}
				return nil
			},
			StatementMaxExecutionTime: func(nodes []int) []v1review.Finding {
				if len(nodes) != stmtCount {
					t.Fatalf("expected %d nodes, got %d", stmtCount, len(nodes))
				}
				return nil
			},
		},
	)

	perStmtRule := advisors[storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT]
	batchRule := advisors[storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME]

	checkCtx := Context{
		Rule: &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
			Level: storepb.SQLReviewRule_ERROR,
		},
	}
	checkCtx.InitMemo()

	_, err := perStmtRule.Check(context.Background(), checkCtx)
	require.NoError(t, err)
	require.Equal(t, 1, extractCalls)

	_, err = batchRule.Check(context.Background(), checkCtx)
	require.NoError(t, err)
	require.Equal(t, 1, extractCalls)

	allocs := testing.AllocsPerRun(20, func() {
		_, checkErr := perStmtRule.Check(context.Background(), checkCtx)
		if checkErr != nil {
			t.Fatal(checkErr)
		}
	})
	require.Equal(t, float64(0), allocs)
}

func TestV1EngineUnifiedRunners(t *testing.T) {
	extract := func(Context) ([]V1Stmt[int], error) {
		return []V1Stmt[int]{
			{Node: 0, Text: "UPDATE t SET a = 1;", BaseLine: 0},
			{Node: 1, Text: "DELETE FROM t;", BaseLine: 1},
		}, nil
	}
	e := NewV1Engine(extract, func(string, int) int32 { return 1 }, V1PosContentStart)

	t.Run("StmtNodes and Batch", func(t *testing.T) {
		stmtRule := e.StmtNodes(
			func(nodes []int) []v1review.Finding {
				if nodes[0] == 1 {
					return []v1review.Finding{{Code: v1review.Truncate, Table: "t"}}
				}
				return []v1review.Finding{{Code: "IGNORE"}}
			},
			func(_ V1Stmt[int], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != v1review.Truncate {
					return 0, "", false
				}
				return code.StatementDisallowTruncate, "blocked " + f.Table, true
			},
		)
		advice, err := stmtRule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, Level: storepb.SQLReviewRule_ERROR},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, "blocked t", advice[0].Content)
		require.Equal(t, int32(2), advice[0].GetStartPosition().GetLine())

		batchRule := e.Batch(
			func(_ Context, nodes []int) []v1review.Finding {
				return []v1review.Finding{{Statement: len(nodes) - 1, Name: "max_execution_time"}}
			},
			func(_ []V1Stmt[int], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				return code.StatementNoMaxExecutionTime, "missing " + f.Name, nil, true
			},
		)
		advice, err = batchRule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, Level: storepb.SQLReviewRule_WARNING},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, "missing max_execution_time", advice[0].Content)
		require.Nil(t, advice[0].StartPosition)
	})

	t.Run("RowLimit positiveOnly and budget cap", func(t *testing.T) {
		manyEngine := NewV1Engine(
			func(Context) ([]V1Stmt[int], error) {
				stmts := make([]V1Stmt[int], common.MaximumLintExplainSize+2)
				for i := range stmts {
					stmts[i] = V1Stmt[int]{Node: i, Text: fmt.Sprintf("UPDATE t SET a = %d;", i), BaseLine: i}
				}
				return stmts, nil
			},
			func(string, int) int32 { return 1 },
			V1PosContentStart,
		)
		rowRule := manyEngine.RowLimit(V1RowLimitConfig[int]{
			Probes: func(nodes []int) []int { return nodes },
			Judge: func(_ []int, probes map[int]v1review.Probe, maxRow int) []v1review.Finding {
				var out []v1review.Finding
				for i, p := range probes {
					if p.Rows > int64(maxRow) {
						out = append(out, v1review.Finding{Statement: i, Code: v1review.RowLimitExceeded, Value: fmt.Sprint(p.Rows)})
					}
				}
				return out
			},
			ExceededCode: code.StatementAffectedRowExceedsLimit,
			ExceededVerb: "affected",
			PositiveOnly: true,
			Estimate: func(_ context.Context, _ Context, _ V1Stmt[int], node int, _ []string) (v1review.Probe, error) {
				if node == 0 {
					return v1review.Probe{Rows: 100}, nil
				}
				return v1review.Probe{Rows: 1}, nil
			},
		})
		advice, err := rowRule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
				Level:   storepb.SQLReviewRule_ERROR,
				Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 0}},
			},
		})
		require.NoError(t, err)
		require.Empty(t, advice)

		advice, err = rowRule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
				Level:   storepb.SQLReviewRule_ERROR,
				Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 10}},
			},
		})
		require.NoError(t, err)
		require.Len(t, advice, 2)
		require.Equal(t, storepb.Advice_ERROR, advice[0].Status)
		require.Contains(t, advice[0].Content, "affected 100 rows (estimated)")
		require.Equal(t, storepb.Advice_WARNING, advice[1].Status)
		require.Contains(t, advice[1].Content, "2 more were not checked")
	})

	t.Run("DMLDryRun requires prober and respects cap", func(t *testing.T) {
		dryRun := e.DMLDryRun(V1DMLDryRunConfig[int]{
			Probes: func(nodes []int) []int { return nodes },
			Judge: func(_ []int, probes map[int]v1review.Probe) []v1review.Finding {
				var out []v1review.Finding
				for i, p := range probes {
					if p.Error != "" {
						out = append(out, v1review.Finding{Statement: i, Value: p.Error})
					}
				}
				return out
			},
		})
		_, err := dryRun.Check(context.Background(), Context{
			DBType: storepb.Engine_MYSQL,
			Rule:   &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "cannot probe the database")

		prober := &ProbeRecorder{}
		advice, err := dryRun.Check(context.Background(), Context{
			DBType: storepb.Engine_MYSQL,
			Prober: prober,
			Rule:   &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING},
		})
		require.NoError(t, err)
		require.Empty(t, advice)
		require.Len(t, prober.Probes, 2)
	})
}
