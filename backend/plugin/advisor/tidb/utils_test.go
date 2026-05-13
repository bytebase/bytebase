package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	omniast "github.com/bytebase/omni/tidb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

// TestGetTiDBOmniNodesCachesAcrossRules pins the Phase 1.5 single-parse-per-
// review invariant: when multiple migrated advisors call getTiDBOmniNodes
// with the same advisor.Context, omni parsing happens once and the result
// slice is reused. Without this, parse cost scales with migrated-advisor
// count and review latency degrades monotonically across the migration
// window. See plans/2026-04-23-omni-tidb-completion-plan.md §1.5.0.
func TestGetTiDBOmniNodesCachesAcrossRules(t *testing.T) {
	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "CREATE TABLE t (id INT)",
					Start: &storepb.Position{Line: 1},
				},
			},
		},
	}
	ctx.InitMemo()

	first, err := getTiDBOmniNodes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, first, "first call should return a parsed statement")

	second, err := getTiDBOmniNodes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, second)

	// Cache hit: same backing array returned both calls. Address equality
	// on the first element proves the slice header was reused, not a fresh
	// re-parse that would allocate a new []OmniStmt.
	require.True(t, &first[0] == &second[0],
		"expected cached []OmniStmt; got fresh re-parse — single-parse-per-review invariant broken")
}

// TestGetTiDBOmniNodesSoftFailsOnGrammarGap pins the Phase 1.5 soft-fail
// invariant: a statement that fails to parse with omni is logged and
// skipped, not propagated as an error that breaks the advisor. The review
// continues with whatever statements omni did parse.
//
// This test feeds a statement that is valid TiDB SQL but uses a deferred
// Phase 2 grammar feature (BATCH ... DRY RUN ...) that omni/tidb does not
// yet support. The expectation is that getTiDBOmniNodes returns the
// successfully-parsed statements and skips the BATCH one, with no error.
func TestGetTiDBOmniNodesSoftFailsOnGrammarGap(t *testing.T) {
	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "CREATE TABLE t (id INT)",
					Start: &storepb.Position{Line: 1},
				},
			},
			{
				Statement: base.Statement{
					Text:  "BATCH ON id LIMIT 5000 UPDATE t SET v = v + 1",
					Start: &storepb.Position{Line: 2},
				},
			},
			{
				Statement: base.Statement{
					Text:  "INSERT INTO t (id) VALUES (1)",
					Start: &storepb.Position{Line: 3},
				},
			},
		},
	}
	ctx.InitMemo()

	got, err := getTiDBOmniNodes(ctx)
	require.NoError(t, err, "soft-fail invariant: parse errors must not propagate")
	require.GreaterOrEqual(t, len(got), 2,
		"expected at least the two non-BATCH statements to parse and be returned")
}

// TestGetTiDBNodesSoftFailsOnBridgeMiss pins the post-flip soft-fail
// contract on the un-migrated-advisor side: when an OmniAST's
// AsPingCapAST() returns (nil, false) — i.e. omni accepted the statement
// but pingcap rejected the same text — getTiDBNodes must SKIP that
// statement, not abort the rule with "AST type mismatch".
//
// Codex caught this on PR #20179: without the soft-fail, post-flip every
// un-migrated advisor emits "Rule check failed: AST type mismatch" for
// statements omni accepts but pingcap doesn't, breaking the migration
// continuity the bridge is meant to preserve.
func TestGetTiDBNodesSoftFailsOnBridgeMiss(t *testing.T) {
	// Pingcap rejects this; omni doesn't matter for this test — what
	// matters is that the bridge returns (nil, false) and getTiDBNodes
	// must skip rather than error.
	bridgeMiss := &tidbparser.OmniAST{
		Text:          "SELECT FROM WHERE;", // pingcap-invalid
		StartPosition: &storepb.Position{Line: 1},
	}
	bridgeHit := &tidbparser.OmniAST{
		Text:          "SELECT 1",
		StartPosition: &storepb.Position{Line: 2},
	}

	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "SELECT FROM WHERE;",
					Start: &storepb.Position{Line: 1},
				},
				AST: bridgeMiss,
			},
			{
				Statement: base.Statement{
					Text:  "SELECT 1",
					Start: &storepb.Position{Line: 2},
				},
				AST: bridgeHit,
			},
		},
	}

	got, err := getTiDBNodes(ctx)
	require.NoError(t, err,
		"bridge miss must be soft-failed (skip), not surfaced as 'AST type mismatch' — see Phase 1.5 invariant #2")
	require.Len(t, got, 1,
		"expected 1 statement from the bridge-hit branch; the bridge-miss statement should be skipped silently")
}

// TestCollectFKCreateTableNilConstraintGuard pins the defensive nil-check
// in collectFKCreateTable. The original batch-4 form passed
// `ostmt.AbsoluteLine(constraint.Loc.Start)` as the line argument to
// buildFKMetaData, which evaluated the dereference BEFORE the helper's
// internal nil check could fire — a nil entry in n.Constraints would
// panic. Per Codex round-1 review on PR #20204; sibling collectFKAlterTable
// was safe because it precomputes stmtLine outside the loop, and
// collectIndexCreateTable / collectUKCreateTable already nil-checked.
//
// Omni's parser doesn't currently produce nil entries from valid SQL, so
// the YAML fixture suite can't reach this branch. The synthetic AST here
// pins the contract: a future change that re-orders the loop body or
// removes the guard fails here.
func TestCollectFKCreateTableNilConstraintGuard(t *testing.T) {
	stmt := &omniast.CreateTableStmt{
		Table: &omniast.TableRef{Name: "t"},
		Constraints: []*omniast.Constraint{
			nil, // would panic on AbsoluteLine(constraint.Loc.Start) without the guard
			{
				Type:       omniast.ConstrForeignKey,
				Name:       "fk_t_author_id_author_id",
				Columns:    []string{"author_id"},
				RefTable:   &omniast.TableRef{Name: "author"},
				RefColumns: []string{"id"},
			},
		},
	}
	ostmt := OmniStmt{
		Node:     stmt,
		Text:     "CREATE TABLE t (author_id INT, FOREIGN KEY (author_id) REFERENCES author(id))",
		BaseLine: 0,
	}

	require.NotPanics(t, func() {
		result := collectFKCreateTable(ostmt, stmt)
		// The non-nil FK constraint must still be collected — the nil guard
		// is `continue`, not an early exit.
		require.Len(t, result, 1, "non-nil FK constraint should still be collected after the nil guard skips the nil entry")
		require.Equal(t, "fk_t_author_id_author_id", result[0].indexName)
	})
}

// TestRunNamingConventionRuleEmitsInternalErrorAdvice pins the
// internal-error advice path in the runNamingConventionRule helper —
// the path that fires when getTemplateRegexp's regexp.Compile fails on
// the post-substitution template. The YAML fixture suite can't reach
// this branch because every fixture uses a valid template; without this
// test, an accidental swap of cfg.internalErrorTitle vs cfg.typeNoun
// (or a regression in the advice's Status/Code/Content shape) would go
// unnoticed.
//
// The test feeds a Format containing an unbalanced paren — no template
// tokens (so payload-token validation passes), but regexp.Compile fails
// after the no-op substitution.
func TestRunNamingConventionRuleEmitsInternalErrorAdvice(t *testing.T) {
	ctx := advisor.Context{
		Rule: &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_NAMING_INDEX_IDX,
			Level: storepb.SQLReviewRule_WARNING,
			Payload: &storepb.SQLReviewRule_NamingPayload{
				NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{
					Format:    "(unbalanced", // valid (no tokens) but fails regexp.Compile
					MaxLength: 64,
				},
			},
		},
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "CREATE INDEX idx_x ON t (a)",
					Start: &storepb.Position{Line: 1},
				},
			},
		},
	}
	ctx.InitMemo()

	advices, err := runNamingConventionRule(ctx, namingRuleConfig{
		mismatchCode:       code.NamingIndexConventionMismatch,
		typeNoun:           "Index",
		internalErrorTitle: "Internal error for index naming convention rule",
	}, func(ostmt OmniStmt) []*indexMetaData {
		// Synthetic collector: surface one indexMetaData per parsed
		// statement so the helper proceeds to regex compilation.
		n, ok := ostmt.Node.(*omniast.CreateIndexStmt)
		if !ok || n.Table == nil {
			return nil
		}
		return []*indexMetaData{{
			indexName: n.IndexName,
			tableName: n.Table.Name,
			metaData: map[string]string{
				advisor.ColumnListTemplateToken: "a",
				advisor.TableNameTemplateToken:  n.Table.Name,
			},
			line: ostmt.FirstTokenLine(),
		}}
	})

	require.NoError(t, err, "regex-compile failure must surface as a per-finding internal-error advice, not a rule-level error")
	require.Len(t, advices, 1, "expected exactly 1 internal-error advice")
	require.Equal(t, "Internal error for index naming convention rule", advices[0].Title,
		"Title must come from cfg.internalErrorTitle, not cfg.typeNoun or the rule.Type string")
	require.Equal(t, code.Internal.Int32(), advices[0].Code,
		"internal-error advices must use code.Internal, not the per-rule mismatch code")
	require.Equal(t, storepb.Advice_WARNING, advices[0].Status)
	require.Contains(t, advices[0].Content, "meet internal error",
		"content shape preserved from pre-extraction inline form")
}

// TestOmniColumnHasComment_PresentAbsentDistinction pins the structural
// contract that omniColumnHasComment uses to distinguish "no COMMENT
// clause" from `COMMENT ”` (deliberately empty). Both produce
// `ColumnDef.Comment == ""`, so the value field alone is ambiguous; the
// signal lives in `Constraints[ColConstrComment]`. If a future omni
// parser change stops emitting the marker for the empty-explicit form,
// the column-comment-required check would silently treat
// deliberately-empty comments as missing. This test catches that.
//
// Per peer review on PR #20217 — "the structural-marker contract is
// plausible but unverified for COMMENT ” specifically". Pin it.
func TestOmniColumnHasComment_PresentAbsentDistinction(t *testing.T) {
	// Case 1: no COMMENT clause at all → omniColumnHasComment returns false.
	noCommentCol := &omniast.ColumnDef{
		Name:        "c",
		Comment:     "",
		Constraints: nil,
	}
	require.False(t, omniColumnHasComment(noCommentCol),
		"absent COMMENT clause: no ColConstrComment marker, omniColumnHasComment must return false")

	// Case 2: explicit COMMENT '' (deliberately empty) → marker present,
	// value empty. omniColumnHasComment returns true.
	emptyCommentCol := &omniast.ColumnDef{
		Name:    "c",
		Comment: "",
		Constraints: []*omniast.ColumnConstraint{
			{Type: omniast.ColConstrComment},
		},
	}
	require.True(t, omniColumnHasComment(emptyCommentCol),
		"explicit COMMENT '' (empty value, marker present): omniColumnHasComment must return true")

	// Case 3: explicit COMMENT 'something' → marker present, value populated.
	regularCommentCol := &omniast.ColumnDef{
		Name:    "c",
		Comment: "hello",
		Constraints: []*omniast.ColumnConstraint{
			{Type: omniast.ColConstrComment},
		},
	}
	require.True(t, omniColumnHasComment(regularCommentCol),
		"explicit COMMENT with value: omniColumnHasComment must return true")
}

// TestOmniIsTimeType pins the DATETIME/TIMESTAMP detection for
// column_current_time_count_limit (batch 13). Pingcap dispatched
// on mysql.TypeDatetime/TypeTimestamp — distinct type bytes,
// no unification concern.
func TestOmniIsTimeType(t *testing.T) {
	cases := []struct {
		name string
		dt   *omniast.DataType
		want bool
	}{
		{"DATETIME", &omniast.DataType{Name: "DATETIME"}, true},
		{"TIMESTAMP", &omniast.DataType{Name: "TIMESTAMP"}, true},
		{"lowercase datetime", &omniast.DataType{Name: "datetime"}, true},
		{"DATE", &omniast.DataType{Name: "DATE"}, false},
		{"TIME", &omniast.DataType{Name: "TIME"}, false},
		{"YEAR", &omniast.DataType{Name: "YEAR"}, false},
		{"VARCHAR", &omniast.DataType{Name: "VARCHAR"}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, omniIsTimeType(tc.dt))
		})
	}
}

// TestOmniIsCurrentTimeFuncCall covers the CURRENT_TIMESTAMP synonym
// detection used by column_current_time_count_limit. Pingcap used
// FnName.L (lowercased); omni keeps user case in FuncCallExpr.Name,
// so we compare case-insensitively.
func TestOmniIsCurrentTimeFuncCall(t *testing.T) {
	cases := []struct {
		name string
		expr omniast.ExprNode
		want bool
	}{
		{"CURRENT_TIMESTAMP uppercase", &omniast.FuncCallExpr{Name: "CURRENT_TIMESTAMP"}, true},
		{"current_timestamp lowercase", &omniast.FuncCallExpr{Name: "current_timestamp"}, true},
		{"NOW", &omniast.FuncCallExpr{Name: "NOW"}, true},
		{"LOCALTIME", &omniast.FuncCallExpr{Name: "LOCALTIME"}, true},
		{"LOCALTIMESTAMP", &omniast.FuncCallExpr{Name: "LOCALTIMESTAMP"}, true},
		{"UTC_TIMESTAMP (not synonym)", &omniast.FuncCallExpr{Name: "UTC_TIMESTAMP"}, false},
		{"unknown function", &omniast.FuncCallExpr{Name: "FOO"}, false},
		{"nil expr", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, omniIsCurrentTimeFuncCall(tc.expr))
		})
	}
}

// TestOmniIsRandFuncCall pins the case-insensitive RAND function
// detection contract used by insert_disallow_order_by_rand. Pingcap-tidb
// matched via lowercase-canonical `FnName.L == ast.Rand`; omni preserves
// the user's case in `FuncCallExpr.Name`, so we compare via EqualFold.
// RAND accepts an optional seed arg — we match regardless of arity.
func TestOmniIsRandFuncCall(t *testing.T) {
	cases := []struct {
		name string
		expr omniast.ExprNode
		want bool
	}{
		{"RAND uppercase", &omniast.FuncCallExpr{Name: "RAND"}, true},
		{"rand lowercase", &omniast.FuncCallExpr{Name: "rand"}, true},
		{"Rand titlecase", &omniast.FuncCallExpr{Name: "Rand"}, true},
		{"RAND with seed arg", &omniast.FuncCallExpr{Name: "RAND", Args: []omniast.ExprNode{nil}}, true},
		{"RANDOM (not synonym)", &omniast.FuncCallExpr{Name: "RANDOM"}, false},
		{"NOT_RAND prefix-like", &omniast.FuncCallExpr{Name: "NOT_RAND"}, false},
		{"unknown function", &omniast.FuncCallExpr{Name: "FOO"}, false},
		{"nil expr", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, omniIsRandFuncCall(tc.expr))
		})
	}
}

// TestOmniIsCharOrBinaryType pins cumulative #22 — pingcap's
// `mysql.TypeString` covered BOTH CHAR and BINARY via charset
// distinction. The omni port must match both names; a mechanical
// port matching only "CHAR" (like the mysql analog does) silently
// drops BINARY coverage. Same shape as cumulative #18 (BLOB/TEXT
// under TypeBlob) and #20 (TINYINT/BOOLEAN under TypeTiny).
//
// VARCHAR/VARBINARY are TypeVarString in pingcap, NOT TypeString —
// pingcap rule did NOT fire on those. Pin the negative case too.
func TestOmniIsCharOrBinaryType(t *testing.T) {
	cases := []struct {
		name string
		dt   *omniast.DataType
		want bool
	}{
		{"CHAR", &omniast.DataType{Name: "CHAR"}, true},
		{"BINARY", &omniast.DataType{Name: "BINARY"}, true},
		{"lowercase char", &omniast.DataType{Name: "char"}, true},
		{"lowercase binary", &omniast.DataType{Name: "binary"}, true},
		{"VARCHAR (negative)", &omniast.DataType{Name: "VARCHAR"}, false},
		{"VARBINARY (negative)", &omniast.DataType{Name: "VARBINARY"}, false},
		{"TEXT", &omniast.DataType{Name: "TEXT"}, false},
		{"BLOB", &omniast.DataType{Name: "BLOB"}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, omniIsCharOrBinaryType(tc.dt))
		})
	}
}

// TestOmniCharLength pins the MySQL default-1 application for bare
// CHAR / BINARY columns. Pingcap's column.Tp.GetFlen() returned the
// canonical default (1) for bare CHAR; omni leaves Length=0, so the
// helper must apply the default explicitly to preserve pingcap
// behavior on column_maximum_character_length.
func TestOmniCharLength(t *testing.T) {
	cases := []struct {
		name string
		dt   *omniast.DataType
		want int
	}{
		{"bare CHAR → 1", &omniast.DataType{Name: "CHAR"}, 1},
		{"bare BINARY → 1", &omniast.DataType{Name: "BINARY"}, 1},
		{"CHAR(10)", &omniast.DataType{Name: "CHAR", Length: 10}, 10},
		{"BINARY(16)", &omniast.DataType{Name: "BINARY", Length: 16}, 16},
		{"CHAR(255)", &omniast.DataType{Name: "CHAR", Length: 255}, 255},
		{"VARCHAR(255) → 0", &omniast.DataType{Name: "VARCHAR", Length: 255}, 0},
		{"INT → 0", &omniast.DataType{Name: "INT"}, 0},
		{"nil → 0", nil, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, omniCharLength(tc.dt))
		})
	}
}
