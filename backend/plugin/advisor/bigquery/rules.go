// Package bigquery implements the SQL advisor rules for BigQuery.
package bigquery

import (
	"fmt"
	"strings"

	omniast "github.com/bytebase/omni/googlesql/ast"
	googlesqlv1review "github.com/bytebase/omni/googlesql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Registers the BigQuery parser so ParseStatements resolves the engine.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/bigquery"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql"
)

func init() {
	extract := advisor.NodeExtractor(func(a base.AST) (omniast.Node, bool) {
		ast, ok := a.(*googlesql.AST)
		if !ok || ast.Node == nil {
			return nil, false
		}
		return ast.Node, true
	})
	byteOffsetLine := func(sql string, offset int) int32 {
		if offset <= 0 {
			return 1
		}
		if offset > len(sql) {
			offset = len(sql)
		}
		return int32(1 + strings.Count(sql[:offset], "\n"))
	}
	e := advisor.NewV1Engine(extract, byteOffsetLine, advisor.V1PosFindingOrFirstLine)

	advisor.RegisterFunc(storepb.Engine_BIGQUERY, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		e.SimpleStmt(googlesqlv1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere, func(_ advisor.V1Stmt[omniast.Node], f v1review.Finding) string {
			return fmt.Sprintf("WHERE clause is required for %s statement.", f.Name)
		}))
}
