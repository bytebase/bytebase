// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	omniast "github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &MigrationCompatibilityAdvisor{})
}

// MigrationCompatibilityAdvisor is the advisor checking for migration
// compatibility. The judgment lives in omni
// (snowflake/v1review.SchemaBackwardCompatibility); this adapter renders the
// findings as advice.
type MigrationCompatibilityAdvisor struct {
}

// compatibilityAdvice is the advice code and content format V1 reported for
// each omni finding, and whether V1 reported it at WARNING whatever the
// rule's level: a drop of an object the change created IF NOT EXISTS.
var compatibilityAdvice = map[v1review.Code]struct {
	code    code.Code
	format  string
	warning bool
}{
	v1review.IncompatibleDropTable:            {code.CompatibilityDropTable, "Drop table %s may cause incompatibility with the existing data and code", false},
	v1review.IncompatibleDropMaybeNewTable:    {code.CompatibilityDropTable, "Drop table %s may cause incompatibility with the existing data and code", true},
	v1review.IncompatibleDropSchema:           {code.CompatibilityDropSchema, "Drop schema %s may cause incompatibility with the existing data and code", false},
	v1review.IncompatibleDropMaybeNewSchema:   {code.CompatibilityDropSchema, "Drop schema %s may cause incompatibility with the existing data and code", true},
	v1review.IncompatibleDropDatabase:         {code.CompatibilityDropDatabase, "Drop database %s may cause incompatibility with the existing data and code", false},
	v1review.IncompatibleDropMaybeNewDatabase: {code.CompatibilityDropDatabase, "Drop database %s may cause incompatibility with the existing data and code", true},
	v1review.IncompatibleDropColumn:           {code.CompatibilityDropColumn, "Drop column %s may cause incompatibility with the existing data and code", false},
}

// Check checks for migration compatibility.
func (*MigrationCompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// The rule judges a statement against the ones before it, so it takes
	// them all at once and names each finding's statement by index.
	var nodes []omniast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range snowflakev1review.SchemaBackwardCompatibility(nodes, checkCtx.CurrentDatabase) {
		advice, ok := compatibilityAdvice[f.Code]
		if !ok {
			continue
		}
		status := level
		if advice.warning {
			status = storepb.Advice_WARNING
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        status,
			Code:          advice.code.Int32(),
			Title:         title,
			Content:       fmt.Sprintf(advice.format, f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
		})
	}
	return adviceList, nil
}
