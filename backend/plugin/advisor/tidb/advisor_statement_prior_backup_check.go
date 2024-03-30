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
				Code:    advisor.StatementPriorBackupCheck,
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

	checker := &statementDisallowMixDMLChecker{
		level:             level,
		title:             string(ctx.Rule.Type),
		dmlStatementCount: make(map[table]map[string]int),
	}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.line = stmtNode.OriginTextPosition()
		if err := checker.extractNode(stmtNode); err != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.Internal,
				Title:   checker.title,
				Content: fmt.Sprintf("Failed to extract node, error: %s", err),
				Line:    checker.line,
			})
		}
	}

	for table, dmlCount := range checker.dmlStatementCount {
		if len(dmlCount) > 1 {
			content := "Found"
			for _, t := range []string{"DELETE", "INSERT", "UPDATE"} {
				count, ok := dmlCount[t]
				if ok {
					content += fmt.Sprintf(" %d %s,", count, t)
				}
			}
			content = strings.TrimSuffix(content, ",")
			content += fmt.Sprintf(" on table `%s`.`%s`, disallow mixing different types of DML statements", table.database, table.table)
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementPriorBackupCheck,
				Title:   checker.title,
				Content: content,
			})
		}
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

type statementDisallowMixDMLChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int

	dmlStatementCount map[table]map[string]int
}

type table struct {
	database string
	table    string
}

func (c *statementDisallowMixDMLChecker) extractNode(n ast.Node) error {
	if n == nil {
		return nil
	}
	var tables []table
	var err error
	var dmlType string
	switch n := n.(type) {
	case *ast.InsertStmt:
		tables, err = extractJoin(n.Table.TableRefs)
		dmlType = "INSERT"
	case *ast.DeleteStmt:
		tables, err = extractTableRefs(n.TableRefs)
		dmlType = "DELETE"
	case *ast.UpdateStmt:
		tables, err = extractTableRefs(n.TableRefs)
		dmlType = "UPDATE"
	default:
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to extract table reference")
	}
	for _, t := range tables {
		if _, ok := c.dmlStatementCount[t]; !ok {
			c.dmlStatementCount[t] = make(map[string]int)
		}
		c.dmlStatementCount[t][dmlType]++
	}
	return nil
}

func extractResultSetNode(n ast.ResultSetNode) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	switch n := n.(type) {
	case *ast.SelectStmt:
		return nil, nil
	case *ast.SubqueryExpr:
		return nil, nil
	case *ast.TableSource:
		return extractTableSource(n)
	case *ast.TableName:
		return extractTableName(n)
	case *ast.Join:
		return extractJoin(n)
	case *ast.SetOprStmt:
		return nil, nil
	}
	return nil, nil
}

func extractTableRefs(n *ast.TableRefsClause) ([]table, error) {
	return extractJoin(n.TableRefs)
}

func extractJoin(n *ast.Join) ([]table, error) {
	l, err := extractResultSetNode(n.Left)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract left node in join")
	}
	r, err := extractResultSetNode(n.Right)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract right node in join")
	}
	l = append(l, r...)
	return l, nil
}

func extractTableSource(n *ast.TableSource) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	return extractResultSetNode(n.Source)
}

func extractTableName(n *ast.TableName) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	return []table{
		{
			table:    n.Name.O,
			database: n.Schema.O,
		},
	}, nil
}
