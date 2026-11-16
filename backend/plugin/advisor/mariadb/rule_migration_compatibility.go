package mariadb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward
// compatibility. The judgment lives in omni
// (mariadb/v1review.SchemaBackwardCompatibility); this adapter renders the
// findings as advice.
type CompatibilityAdvisor struct {
}

// compatibilityCodes maps each omni finding to the advice code V1 reported.
var compatibilityCodes = map[v1review.Code]code.Code{
	v1review.IncompatibleDropDatabase:  code.CompatibilityDropDatabase,
	v1review.IncompatibleRenameTable:   code.CompatibilityRenameTable,
	v1review.IncompatibleDropTable:     code.CompatibilityDropTable,
	v1review.IncompatibleRenameColumn:  code.CompatibilityRenameColumn,
	v1review.IncompatibleDropColumn:    code.CompatibilityDropColumn,
	v1review.IncompatibleAddPrimaryKey: code.CompatibilityAddPrimaryKey,
	v1review.IncompatibleAddUniqueKey:  code.CompatibilityAddUniqueKey,
	v1review.IncompatibleAddForeignKey: code.CompatibilityAddForeignKey,
	v1review.IncompatibleAddCheck:      code.CompatibilityAddCheck,
	v1review.IncompatibleAlterCheck:    code.CompatibilityAlterCheck,
	v1review.IncompatibleAlterColumn:   code.CompatibilityAlterColumn,
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
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mariadbv1review.SchemaBackwardCompatibility(nodes) {
		adviceCode, ok := compatibilityCodes[f.Code]
		if !ok {
			continue
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", strings.TrimSpace(stmt.Text)),
			StartPosition: findingPositionBaseLineTwice(stmt, f),
		})
	}
	return adviceList, nil
}
