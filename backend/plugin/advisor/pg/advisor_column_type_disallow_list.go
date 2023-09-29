package pg

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type restriction.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction.
func (*ColumnTypeDisallowListAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &columnTypeDisallowListChecker{
		level:           level,
		title:           string(ctx.Rule.Type),
		typeRestriction: make(map[string]bool),
	}
	for _, tp := range payload.List {
		checker.typeRestriction[strings.ToLower(tp)] = true
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		ast.Walk(checker, stmt)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type columnTypeDisallowListChecker struct {
	adviceList      []advisor.Advice
	level           advisor.Status
	title           string
	text            string
	line            int
	typeRestriction map[string]bool
}

type columnTypeData struct {
	table  string
	column string
	tp     string
	line   int
}

func (checker *columnTypeDisallowListChecker) Visit(in ast.Node) ast.Visitor {
	var columnList []columnTypeData
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, column := range node.ColumnList {
			exist := false
			typeDisallow := ""
			for tp := range checker.typeRestriction {
				if exist = column.Type.EquivalentType(tp); exist {
					typeDisallow = tp
					break
				}
			}
			if exist {
				columnList = append(columnList, columnTypeData{
					table:  node.Name.Name,
					column: column.ColumnName,
					tp:     strings.ToUpper(typeDisallow),
					line:   column.LastLine(),
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, item := range node.AlterItemList {
			switch cmd := item.(type) {
			case *ast.AddColumnListStmt:
				for _, column := range cmd.ColumnList {
					exist := false
					typeDisallow := ""
					for tp := range checker.typeRestriction {
						if exist = column.Type.EquivalentType(tp); exist {
							typeDisallow = tp
							break
						}
					}
					if exist {
						columnList = append(columnList, columnTypeData{
							table:  node.Table.Name,
							column: column.ColumnName,
							tp:     strings.ToUpper(typeDisallow),
							line:   checker.line,
						})
					}
				}
			case *ast.ChangeColumnStmt:
				exist := false
				typeDisallow := ""
				for tp := range checker.typeRestriction {
					if exist = cmd.Column.Type.EquivalentType(tp); exist {
						typeDisallow = tp
						break
					}
				}
				if exist {
					columnList = append(columnList, columnTypeData{
						table:  node.Table.Name,
						column: cmd.Column.ColumnName,
						tp:     strings.ToUpper(typeDisallow),
						line:   checker.line,
					})
				}
			}
		}
	}

	for _, column := range columnList {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.DisabledColumnType,
			Title:   checker.title,
			Content: fmt.Sprintf("Disallow column type %s but column \"%s\".\"%s\" is", column.tp, column.table, column.column),
			Line:    column.line,
		})
	}
	return checker
}
