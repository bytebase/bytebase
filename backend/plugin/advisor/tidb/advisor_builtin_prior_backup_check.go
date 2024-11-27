package tidb

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/opcode"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	maxMixedDMLCount = 5
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLBuiltinPriorBackupCheck, &StatementPriorBackupCheckAdvisor{})
}

// StatementPriorBackupCheckAdvisor is the advisor checking for no mixed DDL and DML.
type StatementPriorBackupCheckAdvisor struct {
}

// Check checks for no mixed DDL and DML.
func (*StatementPriorBackupCheckAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	if ctx.PreUpdateBackupDetail == nil || ctx.ChangeType != storepb.PlanCheckRunConfig_DML {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	var updateStatements []*ast.UpdateStmt
	var deleteStatements []*ast.DeleteStmt

	for _, stmtNode := range root {
		var isDDL bool
		if _, ok := stmtNode.(ast.DDLNode); ok {
			isDDL = true
		}

		if u, ok := stmtNode.(*ast.UpdateStmt); ok {
			updateStatements = append(updateStatements, u)
		}

		if d, ok := stmtNode.(*ast.DeleteStmt); ok {
			deleteStatements = append(deleteStatements, d)
		}

		if isDDL {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: "Prior backup cannot deal with mixed DDL and DML statements",
				Code:    advisor.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{
					Line: int32(stmtNode.OriginTextPosition()),
				},
			})
		}
	}

	if !advisor.DatabaseExists(ctx, extractDatabaseName(ctx.PreUpdateBackupDetail.Database)) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Need database %q to do prior backup but it does not exist", ctx.PreUpdateBackupDetail.Database),
			Code:    advisor.DatabaseNotExists.Int32(),
			StartPosition: &storepb.Position{
				Line: 0,
			},
		})
	}

	if len(updateStatements)+len(deleteStatements) > maxMixedDMLCount && !updateForOneTableWithUnique(ctx.DBSchema, updateStatements, deleteStatements) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: fmt.Sprintf("Prior backup is feasible only with up to %d statements that are either UPDATE or DELETE, or if all UPDATEs target the same table with a PRIMARY or UNIQUE KEY in the WHERE clause", maxMixedDMLCount),
			Code:    advisor.BuiltinPriorBackupCheck.Int32(),
			StartPosition: &storepb.Position{
				Line: 0,
			},
		})
	}

	return adviceList, nil
}

func updateForOneTableWithUnique(dbSchema *storepb.DatabaseSchemaMetadata, updates []*ast.UpdateStmt, deletes []*ast.DeleteStmt) bool {
	if len(deletes) > 0 {
		return false
	}

	var table *table
	for _, update := range updates {
		tables, err := extractTableRefs(update.TableRefs)
		if err != nil {
			slog.Debug("failed to extract table reference", log.BBError(err))
			return false
		}
		if len(tables) != 1 {
			return false
		}
		if table == nil {
			table = &tables[0]
		} else if !equalTable(table, &tables[0]) {
			return false
		}
		if !hasUniqueInWhereClause(dbSchema, update, table) {
			return false
		}
	}

	return true
}

func hasUniqueInWhereClause(dbSchema *storepb.DatabaseSchemaMetadata, update *ast.UpdateStmt, table *table) bool {
	if update.Where == nil {
		return false
	}
	list := extractColumnsInEqualCondition(table, update.Where)
	columnMap := make(map[string]bool)
	for _, column := range list {
		columnMap[strings.ToLower(column)] = true
	}

	if dbSchema != nil {
		for _, schema := range dbSchema.Schemas {
			for _, tableSchema := range schema.Tables {
				if strings.EqualFold(tableSchema.Name, table.table) {
					for _, index := range tableSchema.Indexes {
						if index.Unique || index.Primary {
							exists := true
							for _, column := range index.Expressions {
								if !columnMap[strings.ToLower(column)] {
									exists = false
									break
								}
							}
							if exists {
								return true
							}
						}
					}
				}
			}
		}
	}

	return false
}

func extractColumnsInEqualCondition(table *table, node ast.ExprNode) []string {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.BinaryOperationExpr:
		switch n.Op {
		case opcode.LogicAnd:
			return append(extractColumnsInEqualCondition(table, n.L), extractColumnsInEqualCondition(table, n.R)...)
		case opcode.EQ:
			if isConstant(n.R) {
				return extractColumnsInEqualCondition(table, n.L)
			} else if isConstant(n.L) {
				return extractColumnsInEqualCondition(table, n.R)
			}

			return nil
		default:
			return nil
		}
	case *ast.ColumnNameExpr:
		if n.Name == nil {
			return nil
		}

		if n.Name.Schema.String() != "" && table.database != "" && !strings.EqualFold(n.Name.Schema.String(), table.database) {
			return nil
		}
		if n.Name.Table.String() != "" && table.table != "" && !strings.EqualFold(n.Name.Table.String(), table.table) {
			return nil
		}
		return []string{n.Name.Name.L}
	default:
		return nil
	}
}

func isConstant(n ast.ExprNode) bool {
	switch n.(type) {
	case ast.ValueExpr:
		return true
	default:
		return false
	}
}

func equalTable(t1, t2 *table) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	return t1.database == t2.database && t1.table == t2.table
}

func extractDatabaseName(databaseUID string) string {
	segments := strings.Split(databaseUID, "/")
	return segments[len(segments)-1]
}

type table struct {
	database string
	table    string
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
