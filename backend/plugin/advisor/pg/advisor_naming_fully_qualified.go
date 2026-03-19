package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*FullyQualifiedObjectNameAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED, &FullyQualifiedObjectNameAdvisor{})
}

// FullyQualifiedObjectNameAdvisor is the advisor checking for fully qualified object names.
type FullyQualifiedObjectNameAdvisor struct {
}

// Check checks for fully qualified object names.
func (*FullyQualifiedObjectNameAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &fullyQualifiedObjectNameRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		dbMetadata: checkCtx.DBSchema,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type fullyQualifiedObjectNameRule struct {
	OmniBaseRule

	dbMetadata *storepb.DatabaseSchemaMetadata
}

func (*fullyQualifiedObjectNameRule) Name() string {
	return "naming_fully_qualified"
}

func (r *fullyQualifiedObjectNameRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.checkRangeVar(n.Relation)
	case *ast.CreateSeqStmt:
		r.checkRangeVar(n.Sequence)
	case *ast.CreateTrigStmt:
		r.checkRangeVar(n.Relation)
	case *ast.IndexStmt:
		r.checkRangeVar(n.Relation)
	case *ast.DropStmt:
		r.handleDropStmt(n)
	case *ast.AlterTableStmt:
		r.checkRangeVar(n.Relation)
	case *ast.AlterSeqStmt:
		r.checkRangeVar(n.Sequence)
	case *ast.RenameStmt:
		r.checkRangeVar(n.Relation)
	case *ast.InsertStmt:
		r.checkRangeVar(n.Relation)
	case *ast.UpdateStmt:
		r.checkRangeVar(n.Relation)
	case *ast.SelectStmt:
		r.handleSelectStmt(n)
	default:
	}
}

func (r *fullyQualifiedObjectNameRule) handleDropStmt(n *ast.DropStmt) {
	for _, nameParts := range omniDropObjectNames(n) {
		objName := strings.Join(nameParts, ".")
		if !isFullyQualifiedName(objName) {
			r.addUnqualifiedAdvice(objName)
		}
	}
}

func (r *fullyQualifiedObjectNameRule) handleSelectStmt(n *ast.SelectStmt) {
	if r.dbMetadata == nil {
		return
	}

	schemaNameMap := r.getSchemaNameMapFromPublic()

	for _, rv := range omniCollectFromClauseRangeVars(n.FromClause) {
		tableName := rv.Relname
		// Only check tables that exist in the schema
		if schemaNameMap == nil || schemaNameMap[tableName] {
			if rv.Schemaname == "" {
				r.addUnqualifiedAdvice(tableName)
			}
		}
	}
}

func (r *fullyQualifiedObjectNameRule) checkRangeVar(rv *ast.RangeVar) {
	if rv == nil {
		return
	}
	var objName string
	if rv.Schemaname != "" {
		objName = rv.Schemaname + "." + rv.Relname
	} else {
		objName = rv.Relname
	}
	if !isFullyQualifiedName(objName) {
		r.addUnqualifiedAdvice(objName)
	}
}

func (r *fullyQualifiedObjectNameRule) addUnqualifiedAdvice(objName string) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    int32(code.NamingNotFullyQualifiedName),
		Title:   r.Title,
		Content: fmt.Sprintf("unqualified object name: '%s'", objName),
		StartPosition: &storepb.Position{
			Line:   r.ContentEndLine(),
			Column: 0,
		},
	})
}

// getSchemaNameMapFromPublic creates a map of table names from the database schema.
func (r *fullyQualifiedObjectNameRule) getSchemaNameMapFromPublic() map[string]bool {
	if r.dbMetadata == nil || r.dbMetadata.Schemas == nil {
		return nil
	}
	filterMap := map[string]bool{}
	for _, schema := range r.dbMetadata.Schemas {
		for _, tbl := range schema.Tables {
			filterMap[tbl.Name] = true
		}
		for _, tbl := range schema.ExternalTables {
			filterMap[tbl.Name] = true
		}
	}
	return filterMap
}

// isFullyQualifiedName checks if an object name is fully qualified (contains a dot).
func isFullyQualifiedName(objName string) bool {
	if objName == "" {
		return true
	}
	return strings.Contains(objName, ".")
}
