package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &MigrationCompatibilityAdvisor{})
}

// MigrationCompatibilityAdvisor is the advisor checking for migration
// compatibility. The judgment lives in omni
// (mssql/v1review.SchemaBackwardCompatibility); this adapter renders the
// findings as advice.
type MigrationCompatibilityAdvisor struct {
}

// compatibilityAdvice is the advice code and content format V1 reported for
// each omni finding; a format holding %s prints the finding's Name.
var compatibilityAdvice = map[v1review.Code]struct {
	code   advisorcode.Code
	format string
}{
	v1review.IncompatibleDropTable:            {advisorcode.CompatibilityDropSchema, "Drop table %s may cause incompatibility with the existing data and code"},
	v1review.IncompatibleDropSchema:           {advisorcode.CompatibilityDropSchema, "Drop schema %s may cause incompatibility with the existing data and code"},
	v1review.IncompatibleDropDatabase:         {advisorcode.CompatibilityDropSchema, "Drop database %s may cause incompatibility with the existing data and code"},
	v1review.IncompatibleDropColumn:           {advisorcode.CompatibilityDropSchema, "Drop column %s may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAlterColumn:          {advisorcode.CompatibilityAlterColumn, "Alter COLUMN %s may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAddForeignKeyNoCheck: {advisorcode.CompatibilityAddForeignKey, "Add FOREIGN KEY WITH NO CHECK may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAddCheckNoCheck:      {advisorcode.CompatibilityAddForeignKey, "Add CHECK WITH NO CHECK may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAddPrimaryKey:        {advisorcode.CompatibilityAddPrimaryKey, "Add PRIMARY KEY may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAddUniqueKey:         {advisorcode.CompatibilityAddUniqueKey, "Add UNIQUE KEY may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAddCheck:             {advisorcode.CompatibilityAddCheck, "Add CHECK may cause incompatibility with the existing data and code"},
	v1review.IncompatibleAddForeignKey:        {advisorcode.CompatibilityAddForeignKey, "Add FOREIGN KEY may cause incompatibility with the existing data and code"},
	v1review.IncompatibleRenameTable:          {advisorcode.CompatibilityRenameTable, "sp_rename may cause incompatibility with the existing data and code, and break scripts and stored procedures."},
}

// Check checks for migration compatibility.
func (*MigrationCompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mssqlv1review.SchemaBackwardCompatibility(nodes, checkCtx.CurrentDatabase) {
		advice, ok := compatibilityAdvice[f.Code]
		if !ok {
			continue
		}
		content := advice.format
		if strings.Contains(content, "%s") {
			content = fmt.Sprintf(advice.format, f.Name)
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advice.code.Int32(),
			Title:         title,
			Content:       content,
			StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
		})
	}
	return adviceList, nil
}
