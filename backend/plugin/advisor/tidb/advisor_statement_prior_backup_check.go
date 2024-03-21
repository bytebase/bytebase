package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLStatementPriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
}

// StatementPriorBackupCheckAdvisor is the advisor checking for no mixed DDL and DML.
type StatementPriorBackupCheckAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementPriorBackupCheckAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	var adviceList []advisor.Advice
	if ctx.PreUpdateBackupDetail == nil || ctx.ChangeType != storepb.PlanCheckRunConfig_DML {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
		return adviceList, nil
	}

	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	for _, stmtNode := range root {
		var isDDL bool
		if _, ok := stmtNode.(ast.DDLNode); ok {
			isDDL = true
		}

		if isDDL {
			adviceList = append(adviceList, advisor.Advice{
				Status:  level,
				Title:   title,
				Content: "Prior backup cannot deal with mixed DDL and DML statements",
				Code:    advisor.StatementMixDDLDML,
				Line:    stmtNode.OriginTextPosition(),
			})
		}
	}

	if !databaseExists(ctx.Context, ctx.Driver, extractDatabaseName(ctx.PreUpdateBackupDetail.Database)) {
		adviceList = append(adviceList, advisor.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Need database %q to do prior backup but it does not exist", ctx.PreUpdateBackupDetail.Database),
			Code:    advisor.DatabaseNotExists,
			Line:    0,
		})
	}

	if len(adviceList) == 0 {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return adviceList, nil
}

func extractDatabaseName(databaseUID string) string {
	segments := strings.Split(databaseUID, "/")
	return segments[len(segments)-1]
}

func databaseExists(ctx context.Context, driver *sql.DB, database string) bool {
	if driver == nil {
		return false
	}
	var count int
	if err := driver.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", database).Scan(&count); err != nil {
		return false
	}
	return count > 0
}
