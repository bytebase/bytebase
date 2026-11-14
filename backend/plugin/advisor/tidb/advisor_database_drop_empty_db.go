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
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the
// MySQLDatabaseAllowDropIfEmpty rule. The judgment lives in omni
// (tidb/v1review.DatabaseDropEmptyDatabase); this adapter passes the
// pre-change schema and renders the findings as advice.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.DatabaseDropEmptyDatabase([]ast.Node{stmt.Node}, dropDatabaseSchema(checkCtx.OriginalMetadata, stmt.Node)) {
			advice := &storepb.Advice{
				Status:        level,
				Title:         title,
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			}
			switch f.Code {
			case v1review.DatabaseNotCurrent:
				advice.Code = code.NotCurrentDatabase.Int32()
				advice.Content = fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", f.Name, checkCtx.OriginalMetadata.DatabaseName())
			case v1review.DatabaseNotEmpty:
				advice.Code = code.DatabaseNotEmpty.Int32()
				advice.Content = fmt.Sprintf("Database `%s` is not allowed to drop if not empty", f.Name)
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
	}
	return adviceList, nil
}

// dropDatabaseSchema returns the pre-change schema one statement is judged
// against: the metadata for a DROP DATABASE, and nothing for any other, which
// reads none. The metadata is read unguarded, so nil metadata panics on a
// DROP DATABASE only, and advisor.Check reports the panic as the rule's error.
func dropDatabaseSchema(metadata *model.DatabaseMetadata, node ast.Node) v1review.Schema {
	if _, ok := node.(*ast.DropDatabaseStmt); !ok {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: metadata.GetProto(), ObjectCaseSensitive: metadata.GetIsObjectCaseSensitive()}
}
