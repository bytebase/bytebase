package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
}

// StatementDmlDryRunAdvisor is the advisor checking for DML dry run.
type StatementDmlDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// BYT-8855: Skip DML dry run if there are DDL statements mixed in, because DML
	// statements often reference objects created by DDL statements, causing false positives.
	if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
		return nil, nil
	}

	driver := checkCtx.Driver
	title := checkCtx.Rule.Type.String()
	var advice []*storepb.Advice
	explainCount := 0

	if driver != nil {
		for _, stmt := range checkCtx.ParsedStatements {
			if stmt.AST == nil {
				continue
			}
			node, ok := mysqlparser.GetOmniNode(stmt.AST)
			if !ok {
				continue
			}

			// Only handle DML statements.
			switch node.(type) {
			case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
			default:
				continue
			}

			baseLine := stmt.BaseLine()
			text := strings.TrimRight(strings.TrimSpace(stmt.Text), ";") + ";"
			line := baseLine + int(mysqlparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line)

			explainCount++
			if _, err := advisor.Query(ctx, advisor.QueryContext{}, driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", text)); err != nil {
				advice = append(advice, &storepb.Advice{
					Status:        level,
					Code:          code.StatementDMLDryRunFailed.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", text, err.Error()),
					StartPosition: common.ConvertANTLRLineToPosition(line),
				})
			}

			if explainCount >= common.MaximumLintExplainSize {
				break
			}
		}
	}

	return advice, nil
}
