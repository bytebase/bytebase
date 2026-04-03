package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number limit.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &indexTotalNumberLimitOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		max:           int(numberPayload.Number),
		lineForTable:  make(map[string]int),
		finalMetadata: checkCtx.FinalMetadata,
	}

	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
	return rule.generateAdvice(), nil
}

type indexTotalNumberLimitOmniRule struct {
	OmniBaseRule
	max           int
	lineForTable  map[string]int
	finalMetadata *model.DatabaseMetadata
}

func (*indexTotalNumberLimitOmniRule) Name() string {
	return "IndexTotalNumberLimitRule"
}

func (r *indexTotalNumberLimitOmniRule) OnStatement(node ast.Node) {
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

func (r *indexTotalNumberLimitOmniRule) generateAdvice() []*storepb.Advice {
	type tableName struct {
		name string
		line int
	}
	var tableList []tableName

	for k, v := range r.lineForTable {
		tableList = append(tableList, tableName{
			name: k,
			line: v,
		})
	}
	slices.SortFunc(tableList, func(i, j tableName) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})

	for _, table := range tableList {
		schema := r.finalMetadata.GetSchemaMetadata("")
		if schema == nil {
			continue
		}
		tableInfo := schema.GetTable(table.name)
		if tableInfo != nil && len(tableInfo.GetProto().Indexes) > r.max {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.IndexCountExceedsLimit.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %d", table.name, r.max, len(tableInfo.GetProto().Indexes)),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
	}

	return r.Advice
}

func (r *indexTotalNumberLimitOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	r.lineForTable[tableName] = r.BaseLine + int(r.LocToLine(n.Loc))
}

func (r *indexTotalNumberLimitOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	r.lineForTable[tableName] = r.BaseLine + int(r.LocToLine(n.Loc))
}

func (r *indexTotalNumberLimitOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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
				if col == nil {
					continue
				}
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
						r.lineForTable[tableName] = r.BaseLine + int(r.LocToLine(n.Loc))
					}
				}
			}
		case ast.ATModifyColumn, ast.ATChangeColumn:
			if cmd.Column != nil {
				for _, c := range cmd.Column.Constraints {
					if c.Type == ast.ColConstrPrimaryKey || c.Type == ast.ColConstrUnique {
						r.lineForTable[tableName] = r.BaseLine + int(r.LocToLine(n.Loc))
					}
				}
			}
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil {
				switch cmd.Constraint.Type {
				case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex, ast.ConstrFulltextIndex:
					r.lineForTable[tableName] = r.BaseLine + int(r.LocToLine(n.Loc))
				default:
				}
			}
		default:
		}
	}
}
