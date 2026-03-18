package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// classifyQueryType classifies an omni AST node into a QueryType.
// allSystems indicates whether all referenced tables are system/info_schema tables.
func classifyQueryType(node ast.Node, allSystems bool) (queryType base.QueryType, isExplainAnalyze bool) {
	if node == nil {
		return base.QueryTypeUnknown, false
	}

	switch n := node.(type) {
	// DML statements
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt, *ast.CopyStmt:
		return base.DML, false

	// SELECT: check for INTO clause (becomes DDL)
	case *ast.SelectStmt:
		if hasOmniIntoClause(n) {
			return base.DDL, false
		}
		if allSystems {
			return base.SelectInfoSchema, false
		}
		return base.Select, false

	// SET is treated as safe (Select)
	case *ast.VariableSetStmt:
		return base.Select, false

	// SHOW → info schema
	case *ast.VariableShowStmt:
		return base.SelectInfoSchema, false

	// EXPLAIN: check for ANALYZE option
	case *ast.ExplainStmt:
		if isExplainAnalyzeOmni(n) {
			qt := classifyExplainedQuery(n.Query, allSystems)
			return qt, true
		}
		return base.Explain, false

	// REFRESH MATERIALIZED VIEW is DML
	case *ast.RefreshMatViewStmt:
		return base.DML, false

	// CALL stored procedure is DML
	case *ast.CallStmt:
		return base.DML, false

	// All DDL statements
	case *ast.CreateStmt, *ast.DropStmt, *ast.AlterTableStmt,
		*ast.CreateTableAsStmt, *ast.CreateSeqStmt, *ast.CreateSchemaStmt,
		*ast.CreatedbStmt, *ast.CreateFunctionStmt, *ast.CreateRoleStmt,
		*ast.IndexStmt, *ast.CreateExtensionStmt, *ast.CreateTrigStmt,
		*ast.CreateEventTrigStmt, *ast.CreateDomainStmt, *ast.CreateConversionStmt,
		*ast.CreateCastStmt, *ast.CreateOpClassStmt, *ast.CreateOpFamilyStmt,
		*ast.CreatePolicyStmt, *ast.CreateAmStmt, *ast.CreateTransformStmt,
		*ast.CreateStatsStmt, *ast.CreateTableSpaceStmt,
		*ast.CreateFdwStmt, *ast.CreateForeignServerStmt,
		*ast.CreateForeignTableStmt, *ast.CreatePLangStmt,
		*ast.CreatePublicationStmt, *ast.CreateSubscriptionStmt,
		*ast.CreateUserMappingStmt,
		*ast.ViewStmt,
		*ast.AlterSeqStmt, *ast.AlterDatabaseStmt, *ast.AlterDatabaseSetStmt,
		*ast.AlterFunctionStmt, *ast.AlterRoleStmt, *ast.AlterRoleSetStmt,
		*ast.AlterCollationStmt, *ast.AlterDomainStmt,
		*ast.AlterExtensionStmt, *ast.AlterExtensionContentsStmt,
		*ast.AlterFdwStmt, *ast.AlterForeignServerStmt,
		*ast.AlterOpFamilyStmt, *ast.AlterPolicyStmt,
		*ast.AlterEventTrigStmt, *ast.AlterObjectDependsStmt,
		*ast.AlterObjectSchemaStmt, *ast.AlterOwnerStmt,
		*ast.AlterOperatorStmt, *ast.AlterTypeStmt, *ast.AlterEnumStmt,
		*ast.AlterStatsStmt, *ast.AlterTableSpaceOptionsStmt,
		*ast.AlterSystemStmt, *ast.AlterPublicationStmt,
		*ast.AlterSubscriptionStmt, *ast.AlterUserMappingStmt,
		*ast.CompositeTypeStmt, *ast.AlterDefaultPrivilegesStmt,
		*ast.AlterTSConfigurationStmt, *ast.AlterTSDictionaryStmt,
		*ast.DropdbStmt, *ast.DropRoleStmt, *ast.DropOwnedStmt,
		*ast.DropSubscriptionStmt, *ast.DropUserMappingStmt,
		*ast.TruncateStmt, *ast.CommentStmt,
		*ast.GrantStmt, *ast.GrantRoleStmt,
		*ast.ClusterStmt, *ast.VacuumStmt, *ast.LockStmt,
		*ast.ReindexStmt, *ast.RuleStmt,
		*ast.RenameStmt, *ast.ReassignOwnedStmt,
		*ast.SecLabelStmt,
		*ast.DoStmt, *ast.DiscardStmt,
		*ast.FetchStmt, *ast.ConstraintsSetStmt, *ast.CheckPointStmt,
		*ast.CreateEnumStmt:
		return base.DDL, false

	default:
		return base.QueryTypeUnknown, false
	}
}

// hasOmniIntoClause checks if a SelectStmt has an INTO clause (SELECT INTO).
func hasOmniIntoClause(n *ast.SelectStmt) bool {
	if n == nil {
		return false
	}
	if n.IntoClause != nil {
		return true
	}
	// Check set operations (UNION/INTERSECT/EXCEPT)
	if n.Larg != nil && hasOmniIntoClause(n.Larg) {
		return true
	}
	if n.Rarg != nil && hasOmniIntoClause(n.Rarg) {
		return true
	}
	return false
}

// isExplainAnalyzeOmni checks if an ExplainStmt has the ANALYZE option.
func isExplainAnalyzeOmni(n *ast.ExplainStmt) bool {
	if n.Options == nil {
		return false
	}
	for _, item := range n.Options.Items {
		if de, ok := item.(*ast.DefElem); ok {
			if strings.EqualFold(de.Defname, "analyze") {
				return true
			}
		}
	}
	return false
}

// classifyExplainedQuery returns the QueryType for the query inside EXPLAIN ANALYZE.
func classifyExplainedQuery(query ast.Node, allSystems bool) base.QueryType {
	if query == nil {
		return base.Select
	}
	switch n := query.(type) {
	case *ast.SelectStmt:
		if hasOmniIntoClause(n) {
			return base.DDL
		}
		return base.Select
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
		return base.DML
	case *ast.DeclareCursorStmt:
		return base.Select
	case *ast.CreateTableAsStmt:
		return base.DDL
	case *ast.RefreshMatViewStmt:
		return base.DML
	case *ast.ExecuteStmt:
		return base.Select
	default:
		return base.Select
	}
}
