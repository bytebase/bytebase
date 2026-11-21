package advisor

import (
	"context"
	"testing"

	"github.com/bytebase/omni/v1review"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestMySQLFamilyColumnCommentNilPayload(t *testing.T) {
	stmt := V1Stmt[int]{Node: 1, Text: "CREATE TABLE t (id INT);", BaseLine: 0}
	extract := func(Context) ([]V1Stmt[int], error) {
		return []V1Stmt[int]{stmt}, nil
	}
	byteOffsetLine := func(string, int) int32 { return 1 }

	for _, tidbQuirks := range []bool{false, true} {
		family := NewMySQLFamily(MySQLFamilyOptions[int]{
			Extract:        extract,
			ByteOffsetLine: byteOffsetLine,
			TiDBQuirks:     tidbQuirks,
		})

		checkMissing := family.ColumnComment(func(_ []int, required bool, maxLength int) []v1review.Finding {
			if tidbQuirks {
				require.False(t, required)
				require.Equal(t, -1, maxLength)
				return []v1review.Finding{{Code: v1review.ColumnCommentTooLong, Table: "t", Name: "id"}}
			}
			require.True(t, required)
			require.Equal(t, 0, maxLength)
			return []v1review.Finding{{Code: v1review.ColumnCommentMissing, Table: "t", Name: "id"}}
		})
		_, err := checkMissing.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_COLUMN_COMMENT,
				Level: storepb.SQLReviewRule_WARNING,
			},
		})
		require.ErrorContains(t, err, "comment_convention_payload is required")

		checkEmpty := family.ColumnComment(func([]int, bool, int) []v1review.Finding {
			return nil
		})
		advice, err := checkEmpty.Check(context.Background(), Context{
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
	family := NewMySQLFamily(MySQLFamilyOptions[int]{
		Extract: func(Context) ([]V1Stmt[int], error) {
			extractCalls++
			stmts := make([]V1Stmt[int], stmtCount)
			for i := range stmts {
				stmts[i] = V1Stmt[int]{Node: i, Text: "SELECT 1"}
			}
			return stmts, nil
		},
		ByteOffsetLine: func(string, int) int32 { return 1 },
	})

	perStmtRule := family.StatementDisallowCommit(func(nodes []int) []v1review.Finding {
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		return nil
	})
	batchRule := family.StatementMaxExecutionTime(func(nodes []int) []v1review.Finding {
		if len(nodes) != stmtCount {
			t.Fatalf("expected %d nodes, got %d", stmtCount, len(nodes))
		}
		return nil
	})

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
