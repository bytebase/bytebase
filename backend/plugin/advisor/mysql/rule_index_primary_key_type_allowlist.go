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
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary key type allowlist.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	allowlist := make(map[string]bool)
	for _, tp := range stringArrayPayload.List {
		allowlist[strings.ToLower(tp)] = true
	}

	rule := &indexPrimaryKeyTypeAllowlistOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowlist:        allowlist,
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableColumnTypes),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexPrimaryKeyTypeAllowlistOmniRule struct {
	OmniBaseRule
	allowlist        map[string]bool
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableColumnTypes
}

func (*indexPrimaryKeyTypeAllowlistOmniRule) Name() string {
	return "IndexPrimaryKeyTypeAllowlistRule"
}

func (r *indexPrimaryKeyTypeAllowlistOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *indexPrimaryKeyTypeAllowlistOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
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
			if c.Type == ast.ColConstrPrimaryKey {
				if _, exists := r.allowlist[columnType]; !exists {
					r.AddAdviceAbsolute(&storepb.Advice{
						Status:        r.Level,
						Code:          code.IndexPKType.Int32(),
						Title:         r.Title,
						Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", col.Name, tableName, columnType),
						StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(col.Loc))),
					})
				}
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

func (r *indexPrimaryKeyTypeAllowlistOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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
					if c.Type == ast.ColConstrPrimaryKey {
						if _, exists := r.allowlist[columnType]; !exists {
							r.AddAdviceAbsolute(&storepb.Advice{
								Status:        r.Level,
								Code:          code.IndexPKType.Int32(),
								Title:         r.Title,
								Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", col.Name, tableName, columnType),
								StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
							})
						}
					}
				}
				r.tablesNewColumns.set(tableName, col.Name, columnType)
			}
		case ast.ATModifyColumn:
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				col := cmd.Column
				columnType := omniDataTypeNameCompact(col.TypeName)
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey {
						if _, exists := r.allowlist[columnType]; !exists {
							r.AddAdviceAbsolute(&storepb.Advice{
								Status:        r.Level,
								Code:          code.IndexPKType.Int32(),
								Title:         r.Title,
								Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", col.Name, tableName, columnType),
								StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
							})
						}
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
					if c.Type == ast.ColConstrPrimaryKey {
						if _, exists := r.allowlist[columnType]; !exists {
							r.AddAdviceAbsolute(&storepb.Advice{
								Status:        r.Level,
								Code:          code.IndexPKType.Int32(),
								Title:         r.Title,
								Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", col.Name, tableName, columnType),
								StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
							})
						}
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

func (r *indexPrimaryKeyTypeAllowlistOmniRule) checkConstraint(tableName string, constraint *ast.Constraint, line int32) {
	if constraint.Type != ast.ConstrPrimaryKey {
		return
	}
	for _, columnName := range constraint.Columns {
		columnType, err := r.getPKColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		columnType = strings.ToLower(columnType)
		if _, exists := r.allowlist[columnType]; !exists {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.IndexPKType.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", columnName, tableName, columnType),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(line)),
			})
		}
	}
}

func (r *indexPrimaryKeyTypeAllowlistOmniRule) getPKColumnType(tableName string, columnName string) (string, error) {
	if columnType, ok := r.tablesNewColumns.get(tableName, columnName); ok {
		return columnType, nil
	}
	column := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column != nil {
		return column.GetProto().Type, nil
	}
	return "", errors.Errorf("cannot find the type of `%s`.`%s`", tableName, columnName)
}
