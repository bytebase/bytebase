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

// TestGetTiDBNodesCachesAcrossRules pins the Phase 1.5 single-parse-per-
// review invariant at the post-flip layer: when multiple migrated advisors
// call getTiDBNodes with the same advisor.Context, the type-switch +
// slice construction happens once and the result slice is reused. Without
// this, every migrated advisor pays the type-switch + allocation cost.
//
// Post-flip (§1.5.N+1) the dispatcher already parsed at split time, so
// stmt.AST is populated with *AST values directly. This test feeds
// pre-populated ParsedStatements to mirror the dispatcher's output.
//
// See plans/2026-04-23-omni-tidb-completion-plan.md §1.5.0 invariant #1.
func TestGetTiDBNodesCachesAcrossRules(t *testing.T) {
	ctx := advisor.Context{
		ParsedStatements: []base.ParsedStatement{
			{
				Statement: base.Statement{
					Text:  "CREATE TABLE t (id INT)",
					Start: &storepb.Position{Line: 1},
				},
				AST: &tidbparser.AST{
					Text:          "CREATE TABLE t (id INT)",
					StartPosition: &storepb.Position{Line: 1},
				},
			},
		},
	}
	ctx.InitMemo()

	first, err := getTiDBNodes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, first, "first call should return a parsed statement")

	second, err := getTiDBNodes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, second)

	// Cache hit: same backing array returned both calls. Address equality
	// on the first element proves the slice header was reused, not a fresh
	// type-switch walk that would allocate a new []Stmt.
	require.True(t, &first[0] == &second[0],
		"expected cached []Stmt; got fresh walk — single-parse-per-review invariant broken")
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
				// Post-flip §1.5.N+1: the dispatcher pre-populates AST.
				// Synthetic *AST wrapping a CreateIndexStmt so the
				// collector closure proceeds to regex compilation.
				AST: &tidbparser.AST{
					Node: &omniast.CreateIndexStmt{
						IndexName: "idx_x",
						Table:     &omniast.TableRef{Name: "t"},
					},
					Text:          "CREATE INDEX idx_x ON t (a)",
					StartPosition: &storepb.Position{Line: 1},
				},
			},
		},
	}
	ctx.InitMemo()

	advices, err := runNamingConventionRule(ctx, namingRuleConfig{
		mismatchCode:       code.NamingIndexConventionMismatch,
		typeNoun:           "Index",
		internalErrorTitle: "Internal error for index naming convention rule",
	}, func(ostmt Stmt) []*indexMetaData {
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

// TestIsRandFuncCall pins the case-insensitive RAND function
// detection contract used by insert_disallow_order_by_rand. Pingcap-tidb
// matched via lowercase-canonical `FnName.L == ast.Rand`; omni preserves
// the user's case in `FuncCallExpr.Name`, so we compare via EqualFold.
// RAND accepts an optional seed arg — we match regardless of arity.
func TestIsRandFuncCall(t *testing.T) {
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
			require.Equal(t, tc.want, isRandFuncCall(tc.expr))
		})
	}
}

// A bytebase-side isDDLStmt enumeration was removed earlier because it was
// brittle for grammar omni once rejected. DDL detection now uses omni's
// authoritative ast.IsDDL classifier (omni parses the full grammar), so no
// bytebase enumeration is needed. End-to-end coverage of the
// DDL-detection path is now in TestTiDBPriorBackupCheckAdvisor.
