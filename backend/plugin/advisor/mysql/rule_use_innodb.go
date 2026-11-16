package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var _ advisor.Advisor = (*UseInnoDBAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, &UseInnoDBAdvisor{})
}

// UseInnoDBAdvisor is the advisor checking for using InnoDB engine. The
// judgment lives in omni (mysql/v1review.EngineMySQLUseInnoDB); this adapter
// renders the findings as advice.
type UseInnoDBAdvisor struct {
}

// Check checks for using InnoDB engine.
func (*UseInnoDBAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		text := &BaseRule{StmtText: stmt.Text}
		for _, f := range mysqlv1review.EngineMySQLUseInnoDB([]ast.Node{node}) {
			// A finding with search text is on the line where the statement
			// first holds that text; any other is on the statement's first line.
			line := text.ContentStartLine()
			if f.Search != "" {
				line = text.FindLineByName(f.Search)
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NotInnoDBEngine.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s;\" doesn't use InnoDB engine", text.TrimmedStmtText()),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(line)),
			})
		}
	}
	return adviceList, nil
}
