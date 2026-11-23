package advisor

import (
	"context"
	"errors"
	"testing"

	"github.com/bytebase/omni/v1review"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
		rule := engine.NamingRule(
			func([]int, v1review.NamingConfig) ([]v1review.Finding, error) {
				t.Fatal("should not evaluate findings when payload is nil")
				return nil, nil
			},
			code.NamingTableConventionMismatch,
			"naming_payload is required for naming table rule",
			"table",
			false,
			func(f v1review.Finding) string { return f.Name },
		)
		_, err := rule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_NAMING_TABLE, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "naming_payload is required for naming table rule")
	})
}

type stubPlanProber struct {
	res db.ProbeResult
	err error
}

func (p stubPlanProber) Probe(context.Context, db.ProbeContext, string) (db.ProbeResult, error) {
	return p.res, p.err
}

func TestV1EngineSharedBuilders(t *testing.T) {
	e := NewV1Engine(
		func(Context) ([]V1Stmt[int], error) {
			return []V1Stmt[int]{
				{Node: 0, Text: "CREATE TABLE t (id INT);", BaseLine: 0},
				{Node: 1, Text: "UPDATE t SET id = 1;", BaseLine: 1},
			}, nil
		},
		nil,
		V1PosFindingOrFirstLine,
	)

	t.Run("NamingIdentifierCase", func(t *testing.T) {
		strict := e.NamingIdentifierCase(func(nodes []int, _ v1review.CaseConfig) []v1review.Finding {
			if nodes[0] == 0 {
				return []v1review.Finding{{Name: "BadCol"}}
			}
			return nil
		}, false)
		_, err := strict.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "naming_case_payload is required")

		advice, err := strict.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:    storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE,
				Level:   storepb.SQLReviewRule_WARNING,
				Payload: &storepb.SQLReviewRule_NamingCasePayload{NamingCasePayload: &storepb.SQLReviewRule_NamingCaseRulePayload{Upper: false}},
			},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, `Identifier "BadCol" should be lower case`, advice[0].Content)

		alwaysUpper := e.NamingIdentifierCase(func(nodes []int, _ v1review.CaseConfig) []v1review.Finding {
			if nodes[0] == 0 {
				return []v1review.Finding{{Name: "low"}}
			}
			return nil
		}, true)
		advice, err = alwaysUpper.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, Level: storepb.SQLReviewRule_ERROR},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, `Identifier "low" should be upper case`, advice[0].Content)
	})

	t.Run("ColumnRequired grouping and quoting", func(t *testing.T) {
		rule := e.ColumnRequired(V1ColumnRequiredConfig[int]{
			Fn: func(nodes []int, _ []string) []v1review.Finding {
				if nodes[0] == 0 {
					return []v1review.Finding{{Table: "t", Name: "created_at"}, {Table: "t", Name: "updated_at"}}
				}
				return nil
			},
			QuoteCol: true,
		})
		advice, err := rule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:    storepb.SQLReviewRule_COLUMN_REQUIRED,
				Level:   storepb.SQLReviewRule_ERROR,
				Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"created_at", "updated_at"}}},
			},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, `Table "t" requires columns: "created_at", "updated_at"`, advice[0].Content)
	})

	t.Run("PriorBackup default and override rendering", func(t *testing.T) {
		line1 := &storepb.Position{Line: 1}
		rule := e.PriorBackup(V1PriorBackupConfig[int]{
			Engine:     storepb.Engine_MSSQL,
			DefaultPos: line1,
			Check: func(Context, []V1Stmt[int], []int, v1review.PriorBackupFacts) ([]v1review.Finding, error) {
				return []v1review.Finding{
					{Code: v1review.PriorBackupDatabaseMissing, Name: "bbdataarchive"},
					{Code: v1review.PriorBackupScriptTooLarge, Value: "100"},
				}, nil
			},
			Render: func(_ []V1Stmt[int], _ []int, f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				if f.Code == v1review.PriorBackupScriptTooLarge {
					return code.BuiltinPriorBackupCheck, "custom too large " + f.Value, line1, true
				}
				return 0, "", nil, false
			},
		})
		disabled, err := rule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, Level: storepb.SQLReviewRule_WARNING},
		})
		require.NoError(t, err)
		require.Empty(t, disabled)

		advice, err := rule.Check(context.Background(), Context{
			EnablePriorBackup: true,
			Rule:              &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, Level: storepb.SQLReviewRule_WARNING},
		})
		require.NoError(t, err)
		require.Len(t, advice, 2)
		require.Equal(t, code.DatabaseNotExists.Int32(), advice[0].Code)
		require.Equal(t, int32(1), advice[0].GetStartPosition().GetLine())
		require.Equal(t, "custom too large 100", advice[1].Content)
	})

	t.Run("ExplainPlanRows", func(t *testing.T) {
		_, err := ExplainPlanRows(context.Background(), Context{DBType: storepb.Engine_POSTGRES}, nil, "EXPLAIN SELECT 1", func(string) (int64, error) { return 0, nil })
		require.ErrorContains(t, err, "cannot probe the database")

		p, err := ExplainPlanRows(context.Background(), Context{
			DBType: storepb.Engine_POSTGRES,
			Prober: stubPlanProber{err: errors.New("syntax error")},
		}, nil, "EXPLAIN SELECT 1", func(string) (int64, error) { return 0, nil })
		require.NoError(t, err)
		require.Equal(t, "syntax error", p.Error)

		p, err = ExplainPlanRows(context.Background(), Context{
			DBType: storepb.Engine_POSTGRES,
			Prober: stubPlanProber{res: db.ProbeResult{Columns: []string{"QUERY PLAN"}, Rows: [][]any{{"ok"}}}},
		}, nil, "EXPLAIN SELECT 1", func(plan string) (int64, error) {
			require.Equal(t, "ok", plan)
			return 42, nil
		})
		require.NoError(t, err)
		require.Equal(t, int64(42), p.Rows)
	})

	t.Run("ExplainProbe", func(t *testing.T) {
		okProbe := ExplainProbe(context.Background(), stubPlanProber{}, db.ProbeContext{}, "EXPLAIN SELECT 1")
		require.Empty(t, okProbe.Error)

		errProbe := ExplainProbe(context.Background(), stubPlanProber{err: errors.New("boom")}, db.ProbeContext{Autocommit: true}, "EXPLAIN PLAN FOR SELECT 1")
		require.Equal(t, "boom", errProbe.Error)
	})

	t.Run("StandardCompatEntries default and TemplateNaming variants", func(t *testing.T) {
		compat := e.BackwardCompatibility(
			func(Context, []int) []v1review.Finding {
				return []v1review.Finding{{Statement: 0, Code: v1review.IncompatibleDropTable}}
			},
			nil,
			func(stmt V1Stmt[int], _ v1review.Finding, _ string) string {
				return e.TrimmedText(stmt.Text) + " incompatible"
			},
		)
		advice, err := compat.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, Level: storepb.SQLReviewRule_ERROR},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, code.CompatibilityDropTable.Int32(), advice[0].Code)
		require.Equal(t, "CREATE TABLE t (id INT) incompatible", advice[0].Content)

		tmpl := e.TemplateNaming(V1TemplateNamingConfig[int]{
			FnNoSchema: func(nodes []int, _ v1review.TemplateConfig) ([]v1review.Finding, error) {
				if nodes[0] == 0 {
					return []v1review.Finding{
						{Code: v1review.PatternInvalid, Detail: "bad"},
						{Code: v1review.NameMismatch, Table: "t", Name: "bad_fk", Pattern: "fk_t"},
					}, nil
				}
				return nil, nil
			},
			Code:          code.NamingFKConventionMismatch,
			Kind:          "Foreign key",
			BacktickQuote: true,
			NilInvalidPos: true,
		})
		advice, err = tmpl.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_NAMING_INDEX_FK,
				Level: storepb.SQLReviewRule_WARNING,
				Payload: &storepb.SQLReviewRule_NamingPayload{
					NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "fk_{{referencing_table}}"},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, advice, 2)
		require.Equal(t, "Internal error for foreign key naming convention rule", advice[0].Title)
		require.Equal(t, "Failed to compile regex: bad", advice[0].Content)
		require.Nil(t, advice[0].StartPosition)
		require.Contains(t, advice[1].Content, "Foreign key in table `t` mismatches the naming convention, expect \"fk_t\" but found `bad_fk`")
	})

	t.Run("TableComment shared commentBatch and StmtCtx skip", func(t *testing.T) {
		tblComment := e.TableComment(
			func(Context, []int, *v1review.TableCommentConfig) []v1review.Finding {
				return []v1review.Finding{{Statement: 0, Code: v1review.TableCommentMissing, Name: "t"}}
			},
			BacktickName,
		)
		_, err := tblComment.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_TABLE_COMMENT, Level: storepb.SQLReviewRule_WARNING},
		})
		require.ErrorContains(t, err, "comment_convention_payload is required for table comment rule")

		advice, err := tblComment.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_TABLE_COMMENT,
				Level: storepb.SQLReviewRule_WARNING,
				Payload: &storepb.SQLReviewRule_CommentConventionPayload{
					CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, "Comment is required for table `t`", advice[0].Content)

		skipped := e.StmtCtx(
			func(context.Context, Context, []int) (func([]int) []v1review.Finding, error) { return nil, nil },
			func(Context, V1Stmt[int], v1review.Finding) (code.Code, string, bool) {
				t.Fatal("render should not run when prepareFn returns nil")
				return 0, "", false
			},
		)
		advice, err = skipped.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, Level: storepb.SQLReviewRule_WARNING},
		})
		require.NoError(t, err)
		require.Empty(t, advice)
	})

	t.Run("RowLimit ExceededPrefix, DMLDryRun StatementText, and formatters", func(t *testing.T) {
		rowRule := e.RowLimit(V1RowLimitConfig[int]{
			Probes: func([]int) []int { return []int{1} },
			Judge: func([]int, map[int]v1review.Probe, int) []v1review.Finding {
				return []v1review.Finding{{Statement: 1, Code: v1review.RowLimitExceeded, Value: "50"}}
			},
			ExceededCode:   code.StatementAffectedRowExceedsLimit,
			ExceededVerb:   "affected",
			ExceededPrefix: "The statement ",
			Estimate: func(context.Context, Context, V1Stmt[int], int, []string) (v1review.Probe, error) {
				return v1review.Probe{Rows: 50}, nil
			},
		})
		advice, err := rowRule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{
				Type:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
				Level:   storepb.SQLReviewRule_ERROR,
				Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 10}},
			},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, `The statement "UPDATE t SET id = 1;" affected 50 rows (estimated). The count exceeds 10.`, advice[0].Content)

		prober := &ProbeRecorder{}
		dryRun := e.DMLDryRun(V1DMLDryRunConfig[int]{
			Probes: func([]int) []int { return []int{1} },
			Judge: func([]int, map[int]v1review.Probe) []v1review.Finding {
				return []v1review.Finding{{Statement: 1, Value: "err"}}
			},
			StatementText: func(stmt V1Stmt[int]) string { return e.TrimmedText(stmt.Text) },
		})
		advice, err = dryRun.Check(context.Background(), Context{
			DBType: storepb.Engine_POSTGRES,
			Prober: prober,
			Rule:   &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_ERROR},
		})
		require.NoError(t, err)
		require.Len(t, prober.Probes, 1)
		require.Equal(t, "EXPLAIN UPDATE t SET id = 1", prober.Probes[0].Statement)
		require.Len(t, advice, 1)
		require.Equal(t, `"UPDATE t SET id = 1" dry runs failed: err`, advice[0].Content)

		startPos := &storepb.Position{Line: 7}
		withStart := NewV1Engine(func(Context) ([]V1Stmt[int], error) {
			return []V1Stmt[int]{{Node: 0, Text: "SELECT 1;", Start: startPos}}, nil
		}, nil, V1PosContentStart)
		constRule := withStart.SimpleConst(func([]int) []v1review.Finding { return []v1review.Finding{{}} }, code.StatementSelectAll, "avoid *", withStart.PosStmtStart)
		advice, err = constRule.Check(context.Background(), Context{
			Rule: &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, Level: storepb.SQLReviewRule_ERROR},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, "avoid *", advice[0].Content)
		require.Equal(t, int32(7), advice[0].GetStartPosition().GetLine())

		batchDB := e.SimpleBatchDB(func(_ []int, dbName string) []v1review.Finding {
			return []v1review.Finding{{Statement: 0, Name: dbName}}
		}, code.TableNoPK, "", FmtFindingName("db=%s"))
		advice, err = batchDB.Check(context.Background(), Context{
			CurrentDatabase: "app",
			Rule:            &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_ERROR},
		})
		require.NoError(t, err)
		require.Len(t, advice, 1)
		require.Equal(t, "db=app", advice[0].Content)
	})
}
