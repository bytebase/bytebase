package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexPkTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, &IndexPkTypeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, &IndexPkTypeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, &IndexPkTypeAdvisor{})
}

// IndexPkTypeAdvisor is the advisor checking for correct type of PK.
type IndexPkTypeAdvisor struct {
}

// Check checks for correct type of PK.
func (*IndexPkTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &indexPkTypeOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableColumnTypes),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexPkTypeOmniRule struct {
	OmniBaseRule
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableColumnTypes
}

func (*indexPkTypeOmniRule) Name() string {
	return "IndexPkTypeRule"
}

func (r *indexPkTypeOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *indexPkTypeOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		columnType := r.getIntOrBigIntStr(col.TypeName)
		for _, c := range col.Constraints {
			if c.Type == ast.ColConstrPrimaryKey {
				r.addAdvice(tableName, col.Name, columnType, r.BaseLine+int(r.LocToLine(col.Loc)))
			}
		}
		r.tablesNewColumns.set(tableName, col.Name, columnType)
	}
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		r.checkConstraint(tableName, constraint, r.LocToLine(constraint.Loc))
	}
}

func (r *indexPkTypeOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddColumn:
			for _, col := range omniGetColumnsFromCmd(cmd) {
				if col == nil || col.TypeName == nil {
					continue
				}
				columnType := r.getIntOrBigIntStr(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey {
						r.addAdvice(tableName, col.Name, columnType, r.BaseLine+int(r.LocToLine(n.Loc)))
					}
				}
				r.tablesNewColumns.set(tableName, col.Name, columnType)
			}
		case ast.ATModifyColumn:
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				col := cmd.Column
				columnType := r.getIntOrBigIntStr(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey {
						r.addAdvice(tableName, col.Name, columnType, r.BaseLine+int(r.LocToLine(n.Loc)))
					}
				}
				r.tablesNewColumns.set(tableName, col.Name, columnType)
			}
		case ast.ATChangeColumn:
			r.tablesNewColumns.delete(tableName, cmd.Name)
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				col := cmd.Column
				columnType := r.getIntOrBigIntStr(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey {
						r.addAdvice(tableName, col.Name, columnType, r.BaseLine+int(r.LocToLine(n.Loc)))
					}
				}
				r.tablesNewColumns.set(tableName, col.Name, columnType)
			}
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil {
				r.checkConstraint(tableName, cmd.Constraint, r.LocToLine(n.Loc))
			}
		default:
		}
	}
}

func (r *indexPkTypeOmniRule) checkConstraint(tableName string, constraint *ast.Constraint, line int32) {
	if constraint.Type != ast.ConstrPrimaryKey {
		return
	}
	for _, columnName := range constraint.Columns {
		columnType, err := r.getPKColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		r.addAdvice(tableName, columnName, columnType, r.BaseLine+int(line))
	}
}

func (r *indexPkTypeOmniRule) addAdvice(tableName, columnName, columnType string, lineNumber int) {
	if !strings.EqualFold(columnType, "INT") && !strings.EqualFold(columnType, "BIGINT") {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.IndexPKType.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", tableName, columnName, columnType),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	}
}

func (r *indexPkTypeOmniRule) getPKColumnType(tableName string, columnName string) (string, error) {
	if columnType, ok := r.tablesNewColumns.get(tableName, columnName); ok {
		return columnType, nil
	}
	column := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column != nil {
		return column.GetProto().Type, nil
	}
	return "", errors.Errorf("cannot find the type of `%s`.`%s`", tableName, columnName)
}

func (*indexPkTypeOmniRule) getIntOrBigIntStr(dt *ast.DataType) string {
	if dt == nil {
		return ""
	}
	switch strings.ToUpper(dt.Name) {
	case "INT", "INTEGER":
		return "INT"
	case "BIGINT":
		return "BIGINT"
	default:
		name := strings.ToLower(dt.Name)
		if dt.Length > 0 {
			if dt.Scale > 0 {
				return fmt.Sprintf("%s(%d,%d)", name, dt.Length, dt.Scale)
			}
			return fmt.Sprintf("%s(%d)", name, dt.Length)
		}
		return name
	}
}
