package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &databaseDropEmptyDBOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		originMetadata: checkCtx.OriginalMetadata,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type databaseDropEmptyDBOmniRule struct {
	OmniBaseRule
	originMetadata *model.DatabaseMetadata
}

func (*databaseDropEmptyDBOmniRule) Name() string {
	return "DatabaseDropEmptyDBRule"
}

func (r *databaseDropEmptyDBOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.DropDatabaseStmt)
	if !ok {
		return
	}
	dbName := n.Name
	line := r.BaseLine + int(r.LocToLine(n.Loc))
	if r.originMetadata.DatabaseName() != dbName {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", dbName, r.originMetadata.DatabaseName()),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	} else if !r.originMetadata.HasNoTable() {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DatabaseNotEmpty.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Database `%s` is not allowed to drop if not empty", dbName),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}
