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
	_ advisor.Advisor = (*IndexTypeNoBlobAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
}

// IndexTypeNoBlobAdvisor is the advisor checking for index type no blob.
type IndexTypeNoBlobAdvisor struct {
}

// Check checks for index type no blob.
func (*IndexTypeNoBlobAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &indexTypeNoBlobOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableColumnTypes),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexTypeNoBlobOmniRule struct {
	OmniBaseRule
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableColumnTypes
}

func (*indexTypeNoBlobOmniRule) Name() string {
	return "IndexTypeNoBlobRule"
}

func (r *indexTypeNoBlobOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.CreateIndexStmt:
		r.checkCreateIndex(n)
	default:
	}
}

func (r *indexTypeNoBlobOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		columnType := omniDataTypeNameCompact(col.TypeName)
		for _, c := range col.Constraints {
			if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
				r.addAdvice(tableName, col.Name, columnType, r.LocToLine(col.Loc))
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

func (r *indexTypeNoBlobOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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
				columnType := omniDataTypeNameCompact(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
						r.addAdvice(tableName, col.Name, columnType, r.LocToLine(n.Loc))
					}
				}
				r.tablesNewColumns.set(tableName, col.Name, columnType)
			}
		case ast.ATModifyColumn:
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				col := cmd.Column
				columnType := omniDataTypeNameCompact(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
						r.addAdvice(tableName, col.Name, columnType, r.LocToLine(n.Loc))
					}
				}
				r.tablesNewColumns.set(tableName, col.Name, columnType)
			}
		case ast.ATChangeColumn:
			r.tablesNewColumns.delete(tableName, cmd.Name)
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				col := cmd.Column
				columnType := omniDataTypeNameCompact(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
						r.addAdvice(tableName, col.Name, columnType, r.LocToLine(n.Loc))
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

func (r *indexTypeNoBlobOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.Table == nil {
		return
	}
	if n.Fulltext || n.Spatial {
		return
	}

	tableName := n.Table.Name
	columnList := omniIndexColumns(n.Columns)
	for _, columnName := range columnList {
		columnType, err := r.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		columnType = strings.ToLower(columnType)
		r.addAdvice(tableName, columnName, columnType, r.LocToLine(n.Loc))
	}
}

func (r *indexTypeNoBlobOmniRule) checkConstraint(tableName string, constraint *ast.Constraint, line int32) {
	var columnList []string
	switch constraint.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex:
		columnList = constraint.Columns
	case ast.ConstrForeignKey:
		columnList = constraint.Columns
	default:
		return
	}

	for _, columnName := range columnList {
		columnType, err := r.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		columnType = strings.ToLower(columnType)
		r.addAdvice(tableName, columnName, columnType, line)
	}
}

func (r *indexTypeNoBlobOmniRule) addAdvice(tableName, columnName, columnType string, line int32) {
	if isBlob(columnType) {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.IndexTypeNoBlob.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", tableName, columnName, columnType),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(line)),
		})
	}
}

func isBlob(columnType string) bool {
	switch strings.ToLower(columnType) {
	case "blob", "tinyblob", "mediumblob", "longblob":
		return true
	default:
		return false
	}
}

func (r *indexTypeNoBlobOmniRule) getColumnType(tableName string, columnName string) (string, error) {
	if columnType, ok := r.tablesNewColumns.get(tableName, columnName); ok {
		return columnType, nil
	}
	column := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column != nil {
		return column.GetProto().Type, nil
	}
	return "", errors.Errorf("cannot find the type of `%s`.`%s`", tableName, columnName)
}
