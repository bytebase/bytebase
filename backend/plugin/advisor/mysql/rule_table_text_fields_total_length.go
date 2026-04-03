package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*TableMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH, &TableMaximumVarcharLengthAdvisor{})
}

type TableMaximumVarcharLengthAdvisor struct {
}

func (*TableMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &tableTextFieldsTotalLengthOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		finalMetadata: checkCtx.FinalMetadata,
		maximum:       int(numberPayload.Number),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.GetAdviceList(), nil
}

type tableTextFieldsTotalLengthOmniRule struct {
	OmniBaseRule
	finalMetadata *model.DatabaseMetadata
	maximum       int
}

func (*tableTextFieldsTotalLengthOmniRule) Name() string {
	return "TableTextFieldsTotalLengthRule"
}

func (r *tableTextFieldsTotalLengthOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *tableTextFieldsTotalLengthOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	schema := r.finalMetadata.GetSchemaMetadata("")
	if schema == nil {
		return
	}
	tableInfo := schema.GetTable(tableName)
	if tableInfo == nil {
		return
	}
	total := getTotalTextLength(tableInfo)
	if total > int64(r.maximum) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.IndexCountExceedsLimit.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, r.maximum),
			StartPosition: &storepb.Position{
				Line: r.LocToLine(n.Loc),
			},
		})
	}
}

func (r *tableTextFieldsTotalLengthOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	schema := r.finalMetadata.GetSchemaMetadata("")
	if schema == nil {
		return
	}
	tableInfo := schema.GetTable(tableName)
	if tableInfo == nil {
		return
	}
	total := getTotalTextLength(tableInfo)
	if total > int64(r.maximum) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.TotalTextLengthExceedsLimit.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Table %q total text column length (%d) exceeds the limit (%d).", tableName, total, r.maximum),
			StartPosition: &storepb.Position{
				Line: r.LocToLine(n.Loc),
			},
		})
	}
}

func getTotalTextLength(tableInfo *model.TableMetadata) int64 {
	var total int64
	columns := tableInfo.GetProto().GetColumns()
	for _, column := range columns {
		total += getTextLength(column.Type)
	}
	return total
}

func getTextLength(s string) int64 {
	s = strings.ToLower(s)
	switch s {
	case "char", "binary":
		return 1
	case "tinyblob", "tinytext":
		return 255
	case "blob", "text":
		return 65_535
	case "mediumblob", "mediumtext":
		return 16_777_215
	case "longblob", "longtext":
		return 4_294_967_295
	default:
		re := regexp.MustCompile(`[a-z]+\((\d+)\)`)
		match := re.FindStringSubmatch(s)
		if len(match) >= 2 {
			n, err := strconv.ParseInt(match[1], 10, 64)
			if err == nil {
				return int64(n)
			}
		}
	}
	return 0
}
