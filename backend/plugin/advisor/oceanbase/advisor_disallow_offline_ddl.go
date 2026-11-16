package oceanbase

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

var (
	_ advisor.Advisor = (*DisallowOfflineDdlAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL, &DisallowOfflineDdlAdvisor{})
}

// DisallowOfflineDdlAdvisor is the advisor checking for disallow Offline DDL.
// omni (oceanbase/v1review.StatementDisallowOfflineDDLLookups) names the
// dropped columns whose EXTRA the database must report; this adapter reads
// them, and omni (StatementDisallowOfflineDDL) judges every statement.
type DisallowOfflineDdlAdvisor struct {
}

// Check checks for disallow Offline DDL.
func (*DisallowOfflineDdlAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes := []ast.Node{node}
		extras := make(map[v1review.ColumnRef]string)
		if checkCtx.Driver != nil {
			for _, ref := range oceanbasev1review.StatementDisallowOfflineDDLLookups(nodes, checkCtx.CurrentDatabase) {
				var extra string
				if err := checkCtx.Driver.QueryRowContext(context.Background(),
					"SELECT EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?",
					ref.Database,
					ref.Table,
					ref.Column,
				).Scan(&extra); err == nil {
					extras[ref] = extra
				}
			}
		}
		for _, f := range oceanbasev1review.StatementDisallowOfflineDDL(nodes, checkCtx.CurrentDatabase, extras) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementOfflineDDL.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("%s is an offline DDL operation.", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(lineOfFinding(stmt, f)),
			})
		}
	}
	return adviceList, nil
}
