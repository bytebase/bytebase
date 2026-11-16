package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ProcedureDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, &ProcedureDisallowCreateAdvisor{})
}

// ProcedureDisallowCreateAdvisor is the advisor checking for disallow creating procedure. The judgment
// lives in omni (mysql/v1review.SystemProcedureDisallowCreate); this adapter renders the findings as
// advice.
type ProcedureDisallowCreateAdvisor struct {
}

// Check checks for disallow creating procedure.
func (*ProcedureDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range mysqlv1review.SystemProcedureDisallowCreate([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.DisallowCreateProcedure.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Procedure is forbidden, but \"%s\" creates", strings.TrimSpace(stmt.Text)),
				StartPosition: findingPositionBaseLineTwice(stmt, f),
			})
		}
	}
	return adviceList, nil
}
