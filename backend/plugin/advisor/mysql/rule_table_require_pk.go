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
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	primaryKeyName = "PRIMARY"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableRequirePKOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		tables:           make(map[string]columnSet),
		line:             make(map[string]int),
		originalMetadata: checkCtx.OriginalMetadata,
	}

	// Walk all statements.
	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})

	// Generate advice after walking all statements.
	rule.generateAdviceList()

	return rule.GetAdviceList(), nil
}

type tableRequirePKOmniRule struct {
	OmniBaseRule
	tables           map[string]columnSet
	line             map[string]int
	originalMetadata *model.DatabaseMetadata
}

func (*tableRequirePKOmniRule) Name() string {
	return "TableRequirePKRule"
}

func (r *tableRequirePKOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.DropTableStmt:
		r.checkDropTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *tableRequirePKOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	r.initEmptyTable(tableName)
	r.line[tableName] = r.BaseLine + int(r.LocToLine(n.Loc))

	// Check column-level PRIMARY KEY.
	for _, col := range n.Columns {
		if col == nil {
			continue
		}
		for _, c := range col.Constraints {
			if c.Type == ast.ColConstrPrimaryKey {
				r.tables[tableName] = newColumnSet([]string{col.Name})
			}
		}
	}

	// Check table-level PRIMARY KEY constraint.
	for _, constraint := range n.Constraints {
		if constraint.Type == ast.ConstrPrimaryKey {
			r.tables[tableName] = newColumnSet(constraint.Columns)
		}
	}
}

func (r *tableRequirePKOmniRule) checkDropTable(n *ast.DropTableStmt) {
	for _, ref := range n.Tables {
		if ref != nil {
			delete(r.tables, ref.Name)
		}
	}
}

func (r *tableRequirePKOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	lineNumber := r.BaseLine + int(r.LocToLine(n.Loc))

	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddConstraint:
			if cmd.Constraint != nil && cmd.Constraint.Type == ast.ConstrPrimaryKey {
				r.tables[tableName] = newColumnSet(cmd.Constraint.Columns)
			}
		case ast.ATDropConstraint:
			// DROP PRIMARY KEY
			if strings.EqualFold(cmd.Name, primaryKeyName) || cmd.Name == "" {
				// cmd.Name == "" when it's DROP PRIMARY KEY (no explicit constraint name)
				r.initEmptyTable(tableName)
				r.line[tableName] = lineNumber
			}
		case ast.ATDropIndex:
			if strings.EqualFold(cmd.Name, primaryKeyName) {
				r.initEmptyTable(tableName)
				r.line[tableName] = lineNumber
			}
		case ast.ATAddColumn:
			if cmd.Column != nil {
				r.handleColumnDef(tableName, cmd.Column)
			}
			for _, col := range cmd.Columns {
				r.handleColumnDef(tableName, col)
			}
		case ast.ATChangeColumn:
			if cmd.Column != nil {
				oldColumn := cmd.Name
				newColumn := cmd.Column.Name
				if r.changeColumn(tableName, oldColumn, newColumn) {
					r.line[tableName] = lineNumber
				}
				r.handleColumnDef(tableName, cmd.Column)
			}
		case ast.ATModifyColumn:
			if cmd.Column != nil {
				r.handleColumnDef(tableName, cmd.Column)
			}
		case ast.ATDropColumn:
			if r.dropColumn(tableName, cmd.Name) {
				r.line[tableName] = lineNumber
			}
		default:
		}
	}
}

func (r *tableRequirePKOmniRule) handleColumnDef(tableName string, col *ast.ColumnDef) {
	if col == nil {
		return
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrPrimaryKey {
			r.tables[tableName] = newColumnSet([]string{col.Name})
		}
	}
}

func (r *tableRequirePKOmniRule) generateAdviceList() {
	tableList := r.getTableList()
	for _, tableName := range tableList {
		if len(r.tables[tableName]) == 0 {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableNoPK.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Table `%s` requires PRIMARY KEY", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(r.line[tableName]),
			})
		}
	}
}

func (r *tableRequirePKOmniRule) changeColumn(tableName string, oldColumn string, newColumn string) bool {
	if r.dropColumn(tableName, oldColumn) {
		pk := r.tables[tableName]
		pk[newColumn] = true
		return true
	}
	return false
}

func (r *tableRequirePKOmniRule) dropColumn(tableName string, colName string) bool {
	if _, ok := r.tables[tableName]; !ok {
		pk := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetIndex(primaryKeyName)
		if pk == nil {
			return false
		}
		r.tables[tableName] = newColumnSet(pk.GetProto().GetExpressions())
	}

	pk := r.tables[tableName]
	_, columnInPk := pk[colName]
	delete(r.tables[tableName], colName)
	return columnInPk
}

func (r *tableRequirePKOmniRule) initEmptyTable(name string) {
	r.tables[name] = make(columnSet)
}

func (r *tableRequirePKOmniRule) getTableList() []string {
	var tableList []string
	for tableName := range r.tables {
		tableList = append(tableList, tableName)
	}
	return tableList
}
