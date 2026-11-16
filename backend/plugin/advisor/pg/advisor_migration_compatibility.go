package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward
// compatibility. The judgment lives in omni
// (pg/v1review.SchemaBackwardCompatibility); this adapter renders the
// findings as advice.
type CompatibilityAdvisor struct {
}

// compatibilityCodes maps each omni finding to the advice code V1 reported.
var compatibilityCodes = map[v1review.Code]advisorcode.Code{
	v1review.IncompatibleDropDatabase:  advisorcode.CompatibilityDropDatabase,
	v1review.IncompatibleDropTable:     advisorcode.CompatibilityDropTable,
	v1review.IncompatibleRenameTable:   advisorcode.CompatibilityRenameTable,
	v1review.IncompatibleRenameColumn:  advisorcode.CompatibilityRenameColumn,
	v1review.IncompatibleDropColumn:    advisorcode.CompatibilityDropColumn,
	v1review.IncompatibleAlterColumn:   advisorcode.CompatibilityAlterColumn,
	v1review.IncompatibleAddPrimaryKey: advisorcode.CompatibilityAddPrimaryKey,
	v1review.IncompatibleAddUniqueKey:  advisorcode.CompatibilityAddUniqueKey,
	v1review.IncompatibleAddForeignKey: advisorcode.CompatibilityAddForeignKey,
	v1review.IncompatibleAddCheck:      advisorcode.CompatibilityAddCheck,
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// The rule judges a statement against the ones before it, so it takes
	// them all at once and names each finding's statement by index.
	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.SchemaBackwardCompatibility(nodes) {
		adviceCode, ok := compatibilityCodes[f.Code]
		if !ok {
			continue
		}
		stmt := stmts[f.Statement]
		rule := &BaseRule{StmtText: stmt.Text}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, rule.TrimmedStmtText()),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(rule.ContentStartLine())),
		})
	}
	return adviceList, nil
}
