// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &namingTableNoKeywordChecker{
		level:      level,
		title:      checkCtx.Rule.Type.String(),
		adviceList: []*storepb.Advice{},
	}

	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		checker.checkStmt(node)
	}

	return checker.adviceList, nil
}

type namingTableNoKeywordChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

func (c *namingTableNoKeywordChecker) checkStmt(node snowflakeast.Node) {
	switch n := node.(type) {
	case *snowflakeast.CreateTableStmt:
		// The legacy listener only fired on the plain CREATE TABLE grammar rule;
		// CREATE TABLE ... AS SELECT / LIKE / CLONE were separate rules it did
		// not subscribe to. Mirror that scope exactly.
		if n.AsSelect != nil || n.Like != nil || n.Clone != nil {
			return
		}
		if n.Name == nil {
			return
		}
		c.checkTableName(n.Name.Name)
	case *snowflakeast.AlterTableStmt:
		for _, action := range n.Actions {
			if action.Kind != snowflakeast.AlterTableRename || action.NewName == nil {
				continue
			}
			c.checkTableName(action.NewName.Name)
		}
	default:
	}
}

// checkTableName reports the table part of an object name when its canonical
// (folded) form is a Snowflake keyword. The legacy listener emitted this
// advice without a start position; keep that shape for identical output.
func (c *namingTableNoKeywordChecker) checkTableName(name snowflakeast.Ident) {
	tableName := name.Normalize()
	if snowsqlparser.IsSnowflakeKeyword(tableName, false) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    code.NameIsKeywordIdentifier.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
		})
	}
}
