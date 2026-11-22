package advisor

import (
	"context"
	"testing"

	"github.com/bytebase/omni/v1review"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestV1EnginePrepareMemoizesStmtsAndNodes(t *testing.T) {
	const stmtCount = 100
	extractCalls := 0
	engine := NewV1Engine(
		func(Context) ([]V1Stmt[int], error) {
			extractCalls++
			stmts := make([]V1Stmt[int], stmtCount)
			for i := range stmts {
				stmts[i] = V1Stmt[int]{Node: i, Text: "SELECT 1"}
			}
			return stmts, nil
		},
		func(string, int) int32 { return 1 },
		V1PosContentStart,
	)

	perStmtRule := engine.SimpleStmt(func(nodes []int) []v1review.Finding {
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		return nil
	}, code.StatementSelectAll, func(V1Stmt[int], v1review.Finding) string { return "" })

	batchRule := engine.SimpleBatch(func(_ Context, nodes []int) []v1review.Finding {
		if len(nodes) != stmtCount {
			t.Fatalf("expected %d nodes, got %d", stmtCount, len(nodes))
		}
		return nil
	}, code.TableNoPK, "", func(v1review.Finding) string { return "" })

	checkCtx := Context{
		Rule: &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
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

func TestV1EngineNilPayloads(t *testing.T) {
	engine := NewV1Engine(
		func(Context) ([]V1Stmt[int], error) {
			return []V1Stmt[int]{{Node: 1, Text: "CREATE TABLE t (id INT);"}}, nil
		},
		func(string, int) int32 { return 1 },
		V1PosFinding,
	)

	t.Run("column comment nil payload errors on finding and succeeds on empty", func(t *testing.T) {
		onFinding := engine.ColumnComment(
			func(_ Context, _ []int, required bool, maxLength int) []v1review.Finding {
				require.True(t, required)
				require.Equal(t, 0, maxLength)
				return []v1review.Finding{{Code: v1review.ColumnCommentMissing, Table: "t", Name: "id"}}
			},
			0,
			false,
			func(f v1review.Finding) string { return f.Name },
		)
		_, err := onFinding.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_COLUMN_COMMENT, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "comment_convention_payload is required")

		noFinding := engine.ColumnComment(
			func(Context, []int, bool, int) []v1review.Finding { return nil },
			0,
			false,
			func(f v1review.Finding) string { return f.Name },
		)
		advice, err := noFinding.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_COLUMN_COMMENT, Level: storepb.SQLReviewRule_WARNING},
		})
		require.NoError(t, err)
		require.Empty(t, advice)
	})

	t.Run("string array nil payload returns exact error without warnings", func(t *testing.T) {
		rule := engine.StringArrayStmt(
			func([]int, []string) []v1review.Finding {
				t.Fatal("should not evaluate findings when payload is nil")
				return nil
			},
			"string_array_payload is required for column type disallow list rule",
			code.DisabledColumnType,
			"",
			func(V1Stmt[int], v1review.Finding) string { return "" },
		)
		advice, err := rule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "string_array_payload is required for column type disallow list rule")
		require.Empty(t, advice)
	})

	t.Run("number nil payload returns exact error", func(t *testing.T) {
		rule := engine.NumberStmt(
			func([]int, int) []v1review.Finding {
				t.Fatal("should not evaluate findings when payload is nil")
				return nil
			},
			true,
			code.VarcharLengthExceedsLimit,
			func(V1Stmt[int], v1review.Finding, int) string { return "" },
		)
		_, err := rule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "number_payload is required for this rule")
	})

	t.Run("naming nil payload returns exact error", func(t *testing.T) {
		rule := engine.NamingRegex(
			func([]int, v1review.NamingConfig) ([]v1review.Finding, error) {
				t.Fatal("should not evaluate findings when payload is nil")
				return nil, nil
			},
			code.NamingTableConventionMismatch,
			"naming_payload is required for naming table rule",
			func(f v1review.Finding, _ string) string { return f.Name },
			nil,
		)
		_, err := rule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_NAMING_TABLE, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "naming_payload is required for naming table rule")
	})
}
