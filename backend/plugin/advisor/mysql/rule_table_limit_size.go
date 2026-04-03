package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_LIMIT_SIZE, &MaximumTableSizeAdvisor{})
}

type MaximumTableSizeAdvisor struct {
}

var (
	_ advisor.Advisor = &MaximumTableSizeAdvisor{}
)

// If table size > xx bytes, then warning/error.
func (*MaximumTableSizeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	status, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableLimitSizeOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: status,
			Title: checkCtx.Rule.Type.String(),
		},
		maxRows:    int(numberPayload.Number),
		dbMetadata: checkCtx.DBSchema,
	}

	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})

	// Generate advice based on collected table information.
	rule.generateAdvice()

	return rule.GetAdviceList(), nil
}

type tableLimitSizeOmniRule struct {
	OmniBaseRule
	affectedTabNames []string
	maxRows          int
	dbMetadata       *storepb.DatabaseSchemaMetadata
}

func (*tableLimitSizeOmniRule) Name() string {
	return "TableLimitSizeRule"
}

func (r *tableLimitSizeOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.TruncateStmt:
		r.checkTruncate(n)
	case *ast.DropTableStmt:
		r.checkDropTable(n)
	default:
	}
}

func (r *tableLimitSizeOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	r.affectedTabNames = append(r.affectedTabNames, n.Table.Name)
}

func (r *tableLimitSizeOmniRule) checkTruncate(n *ast.TruncateStmt) {
	for _, ref := range n.Tables {
		if ref != nil {
			r.affectedTabNames = append(r.affectedTabNames, ref.Name)
		}
	}
}

func (r *tableLimitSizeOmniRule) checkDropTable(n *ast.DropTableStmt) {
	for _, ref := range n.Tables {
		if ref != nil {
			r.affectedTabNames = append(r.affectedTabNames, ref.Name)
		}
	}
}

func (r *tableLimitSizeOmniRule) generateAdvice() {
	if r.dbMetadata != nil && len(r.dbMetadata.Schemas) != 0 {
		for _, tabName := range r.affectedTabNames {
			tableRows := getTabRowsByName(tabName, r.dbMetadata.Schemas[0].Tables)
			if tableRows >= int64(r.maxRows) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.TableExceedLimitSize.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Apply DDL on large table '%s' ( %d rows ) will lock table for a long time", tabName, tableRows),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
				})
			}
		}
	}
}

func getTabRowsByName(targetTabName string, tables []*storepb.TableMetadata) int64 {
	for _, table := range tables {
		if table.Name == targetTabName {
			return table.RowCount
		}
	}
	return 0
}
