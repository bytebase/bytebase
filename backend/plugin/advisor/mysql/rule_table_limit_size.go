package mysql

import (
	"context"
	"fmt"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

	rule := &tableLimitSizeRule{
		BaseRule: BaseRule{
			Level: status,
			Title: checkCtx.Rule.Type.String(),
		},
		maxRows:    int(numberPayload.Number),
		dbMetadata: checkCtx.DBSchema,
	}

	RunRules(checkCtx.ParsedStatements, []Rule{rule})

	// Generate advice based on collected table information.
	rule.generateAdvice()

	return rule.GetAdviceList(), nil
}

type tableLimitSizeRule struct {
	BaseRule
	affectedTabNames []string
	maxRows          int
	dbMetadata       *metadatapb.DatabaseSchemaMetadata
}

func (*tableLimitSizeRule) Name() string {
	return "TableLimitSizeRule"
}

func (r *tableLimitSizeRule) OnStatement(node ast.Node) {
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

func (r *tableLimitSizeRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	r.affectedTabNames = append(r.affectedTabNames, n.Table.Name)
}

func (r *tableLimitSizeRule) checkTruncate(n *ast.TruncateStmt) {
	for _, ref := range n.Tables {
		if ref != nil {
			r.affectedTabNames = append(r.affectedTabNames, ref.Name)
		}
	}
}

func (r *tableLimitSizeRule) checkDropTable(n *ast.DropTableStmt) {
	for _, ref := range n.Tables {
		if ref != nil {
			r.affectedTabNames = append(r.affectedTabNames, ref.Name)
		}
	}
}

func (r *tableLimitSizeRule) generateAdvice() {
	if r.dbMetadata != nil && len(r.dbMetadata.Schemas) != 0 {
		for _, tabName := range r.affectedTabNames {
			tableRows := getTabRowsByName(tabName, r.dbMetadata.Schemas[0].Tables)
			if tableRows >= int64(r.maxRows) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.TableExceedLimitSize.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Apply DDL on large table '%s' ( %d rows ) will lock table for a long time", tabName, tableRows),
					StartPosition: base.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
				})
			}
		}
	}
}

func getTabRowsByName(targetTabName string, tables []*metadatapb.TableMetadata) int64 {
	for _, table := range tables {
		if table.Name == targetTabName {
			return table.RowCount
		}
	}
	return 0
}
