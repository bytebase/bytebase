// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"strings"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*TableRequirePkAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePkAdvisor{})
}

// TableRequirePkAdvisor is the advisor checking for table require primary key.
type TableRequirePkAdvisor struct {
}

// Check checks for table require primary key.
func (*TableRequirePkAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableRequirePkRule(level, checkCtx.Rule.Type.String())

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		rule.checkStatement(node, stmt.Text)
	}

	return rule.GetAdviceList(), nil
}

// TableRequirePkRule checks for table require primary key.
type TableRequirePkRule struct {
	BaseRule

	// tableHasPrimaryKey is a map of normalized table name to whether the table has primary key.
	tableHasPrimaryKey map[string]bool
	// tableOriginalName is a map of normalized table name to original table name.
	// The key of the tableOriginalName is the superset of the key of the tableHasPrimaryKey.
	tableOriginalName map[string]string
	// tableLine is a map of normalized table name to the line number of the table.
	// The key of the tableLine is the superset of the key of the tableHasPrimaryKey.
	tableLine map[string]int

	// stmtText is the SQL text of the statement currently being checked; node
	// Loc offsets are relative to it.
	stmtText string
}

// NewTableRequirePkRule creates a new TableRequirePkRule.
func NewTableRequirePkRule(level storepb.Advice_Status, title string) *TableRequirePkRule {
	return &TableRequirePkRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		tableHasPrimaryKey: make(map[string]bool),
		tableOriginalName:  make(map[string]string),
		tableLine:          make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableRequirePkRule) Name() string {
	return "TableRequirePkRule"
}

// GetAdviceList returns the accumulated advice list, generating final advice for tables without PK.
func (r *TableRequirePkRule) GetAdviceList() []*storepb.Advice {
	for tableName, has := range r.tableHasPrimaryKey {
		if !has {
			// tableLine already stores absolute line numbers (baseLine + line at
			// time of encounter), so append directly without re-adding baseLine.
			r.adviceList = append(r.adviceList, &storepb.Advice{
				Status:        r.level,
				Code:          code.TableNoPK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", r.tableOriginalName[tableName]),
				StartPosition: common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			})
		}
	}
	return r.adviceList
}

// checkStatement checks one statement's omni AST node.
func (r *TableRequirePkRule) checkStatement(node omniast.Node, stmtText string) {
	r.stmtText = stmtText
	switch n := node.(type) {
	case *omniast.CreateTableStmt:
		r.checkCreateTable(n)
	case *omniast.DropStmt:
		r.checkDropStmt(n)
	case *omniast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *TableRequirePkRule) checkCreateTable(stmt *omniast.CreateTableStmt) {
	// The legacy advisor only fired on plain CREATE TABLE (the Create_table
	// context); CREATE TABLE ... AS SELECT / LIKE / CLONE parsed as different
	// contexts and were never checked.
	if stmt.AsSelect != nil || stmt.Like != nil || stmt.Clone != nil {
		return
	}

	normalizedTableName := r.normalizedTableName(stmt.Name)

	r.tableHasPrimaryKey[normalizedTableName] = false
	r.tableOriginalName[normalizedTableName] = stmt.Name.String()
	r.tableLine[normalizedTableName] = r.baseLine + r.line(stmt.Loc.Start)

	for _, column := range stmt.Columns {
		if column.InlineConstraint != nil && column.InlineConstraint.Type == omniast.ConstrPrimaryKey {
			r.tableHasPrimaryKey[normalizedTableName] = true
		}
	}
	for _, constraint := range stmt.Constraints {
		if constraint.Type == omniast.ConstrPrimaryKey {
			r.tableHasPrimaryKey[normalizedTableName] = true
		}
	}
}

func (r *TableRequirePkRule) checkDropStmt(stmt *omniast.DropStmt) {
	if stmt.Kind != omniast.DropTable {
		return
	}
	normalizedTableName := r.normalizedTableName(stmt.Name)

	delete(r.tableHasPrimaryKey, normalizedTableName)
	delete(r.tableOriginalName, normalizedTableName)
	delete(r.tableLine, normalizedTableName)
}

func (r *TableRequirePkRule) checkAlterTable(stmt *omniast.AlterTableStmt) {
	// The legacy advisor only tracked ALTER TABLE statements with a constraint
	// action (ADD/DROP/RENAME CONSTRAINT); column actions were ignored.
	if !alterTableHasConstraintActionForPk(stmt) {
		return
	}

	normalizedTableName := r.normalizedTableName(stmt.Name)
	r.tableOriginalName[normalizedTableName] = stmt.Name.String()

	for _, action := range stmt.Actions {
		switch action.Kind {
		case omniast.AlterTableAddConstraint:
			if action.Constraint != nil && action.Constraint.Type == omniast.ConstrPrimaryKey {
				r.tableHasPrimaryKey[normalizedTableName] = true
			}
		case omniast.AlterTableDropConstraint:
			// Only an explicit DROP PRIMARY KEY flips the flag; DROP CONSTRAINT
			// <name> does not reveal the constraint type, matching the legacy
			// advisor.
			if action.IsPrimaryKey {
				if _, ok := r.tableHasPrimaryKey[normalizedTableName]; ok {
					r.tableHasPrimaryKey[normalizedTableName] = false
					r.tableLine[normalizedTableName] = r.baseLine + r.line(action.Loc.Start)
				}
			}
		default:
		}
	}
}

func alterTableHasConstraintActionForPk(stmt *omniast.AlterTableStmt) bool {
	for _, action := range stmt.Actions {
		switch action.Kind {
		case omniast.AlterTableAddConstraint, omniast.AlterTableDropConstraint, omniast.AlterTableRenameConstraint:
			return true
		default:
		}
	}
	return false
}

// normalizedTableName normalizes an object name into the same
// "database.schema.table" key the legacy advisor produced via
// NormalizeSnowSQLObjectName(name, "", "PUBLIC").
func (*TableRequirePkRule) normalizedTableName(name *omniast.ObjectName) string {
	if name == nil {
		return ""
	}
	database := ""
	if d := name.Database.Normalize(); d != "" {
		database = d
	}
	schema := "PUBLIC"
	if s := name.Schema.Normalize(); s != "" {
		schema = s
	}
	parts := []string{database, schema}
	if o := name.Name.Normalize(); o != "" {
		parts = append(parts, o)
	}
	return strings.Join(parts, ".")
}

// line converts a byte offset within the current statement text to a 1-based
// line number, mirroring the ANTLR token line the legacy advisor reported.
func (r *TableRequirePkRule) line(offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(r.stmtText) {
		offset = len(r.stmtText)
	}
	return 1 + strings.Count(r.stmtText[:offset], "\n")
}
