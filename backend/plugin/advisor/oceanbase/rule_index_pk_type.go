package oceanbase

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexPkTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, &IndexPkTypeAdvisor{})
}

// IndexPkTypeAdvisor is the advisor checking for correct type of PK. The
// judgment lives in omni (oceanbase/v1review.IndexPKTypeLimit); this adapter
// passes the script and the pre-change schema and renders the findings as
// advice.
type IndexPkTypeAdvisor struct {
}

// Check checks for correct type of PK.
func (*IndexPkTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	findings, err := oceanbasev1review.IndexPKTypeLimit(nodes, schemaOf(checkCtx.OriginalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.OriginalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexPKType.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", f.Table, f.Name, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}

// scriptNodes returns the statements that have an omni AST and their nodes,
// in order, so a finding's Statement indexes both.
func scriptNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
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
