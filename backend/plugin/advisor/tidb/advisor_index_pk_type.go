package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexPkTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, &IndexPkTypeAdvisor{})
}

// IndexPkTypeAdvisor is the advisor checking for correct type of PK. The
// judgment lives in omni (tidb/v1review.IndexPKTypeLimit); this adapter
// passes the script and the pre-change schema and renders the findings as
// advice.
type IndexPkTypeAdvisor struct {
}

// Check checks for correct type of PK.
func (*IndexPkTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	findings, err := tidbv1review.IndexPKTypeLimit(scriptNodes(stmts), schemaOf(checkCtx.OriginalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.OriginalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexPKType.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", f.Table, f.Name, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(findingLine(stmts[f.Statement], f)),
		})
	}
	return adviceList, nil
}

// scriptNodes returns the statements' nodes in order, so a finding's
// Statement indexes both.
func scriptNodes(stmts []Stmt) []ast.Node {
	nodes := make([]ast.Node, 0, len(stmts))
	for _, stmt := range stmts {
		nodes = append(nodes, stmt.Node)
	}
	return nodes
}

// schemaOf returns a snapshot as omni reads it; nil holds nothing.
func schemaOf(m *model.DatabaseMetadata) v1review.Schema {
	if m == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: m.GetProto(), ObjectCaseSensitive: m.GetIsObjectCaseSensitive()}
}

// readMissingSchema repeats the V1 advisors' unguarded read of a snapshot
// omni found missing, so a nil snapshot panics where it did and
// advisor.Check reports the panic as the rule's error.
func readMissingSchema(m *model.DatabaseMetadata, err error) {
	if err == v1review.ErrSchemaMissing {
		m.GetSchemaMetadata("")
	}
}

// findingLine is the 1-based line of the script a finding lies on: its
// range's start, or the statement's first character that is not white
// space when it has no range.
func findingLine(stmt Stmt, f v1review.Finding) int {
	if f.Start == 0 && f.End == 0 {
		return stmt.FirstTokenLine()
	}
	return stmt.AbsoluteLine(f.Start)
}
