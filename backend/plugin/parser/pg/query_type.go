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
		// A data-modifying CTE writes, so the SELECT needs DML permission.
		if containsWriteCTE(n) {
			return base.DML, false
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
			qt := classifyExplainedQuery(n.Query)
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
	return omniIntoClause(n) != nil
}

// omniIntoClause returns the INTO clause of n, searching set-operation arms
// (UNION/INTERSECT/EXCEPT) — the parser attaches INTO to the first arm, not the root.
func omniIntoClause(n *ast.SelectStmt) *ast.IntoClause {
	if n == nil {
		return nil
	}
	if n.IntoClause != nil {
		return n.IntoClause
	}
	if c := omniIntoClause(n.Larg); c != nil {
		return c
	}
	return omniIntoClause(n.Rarg)
}

// isExplainAnalyzeOmni reports whether an ExplainStmt executes its query: its last ANALYZE option
// is not FALSE, OFF, or 0, the values PostgreSQL reads as false.
func isExplainAnalyzeOmni(n *ast.ExplainStmt) bool {
	if n.Options == nil {
		return false
	}
	analyze := false
	for _, item := range n.Options.Items {
		de, ok := item.(*ast.DefElem)
		if !ok || !strings.EqualFold(de.Defname, "analyze") {
			continue
		}
		switch arg := de.Arg.(type) {
		case *ast.String:
			analyze = !strings.EqualFold(arg.Str, "false") && !strings.EqualFold(arg.Str, "off")
		case *ast.Integer:
			analyze = arg.Ival != 0
		default:
			analyze = true
		}
	}
	return analyze
}

// UnwrapExplainAnalyze returns the statement that an EXPLAIN ANALYZE executes and that statement's text
// within text, the text of the EXPLAIN. It returns any other node and text unchanged.
func UnwrapExplainAnalyze(node ast.Node, text string) (ast.Node, string) {
	explain, ok := node.(*ast.ExplainStmt)
	if !ok || !isExplainAnalyzeOmni(explain) {
		return node, text
	}
	// The statement runs from its WITH clause, which the location of a SELECT leaves out, to the end
	// of the EXPLAIN, which its location can also leave out, as for ORDER BY.
	start := ast.NodeLoc(explain.Query).Start
	if with := getWithClause(explain.Query); with != nil && with.Loc.Start >= 0 && with.Loc.Start < start {
		start = with.Loc.Start
	}
	if start < 0 || start >= explain.Loc.End || explain.Loc.End > len(text) {
		return explain.Query, text
	}
	return explain.Query, text[start:explain.Loc.End]
}

// classifyExplainedQuery returns the QueryType for the query inside EXPLAIN ANALYZE.
func classifyExplainedQuery(query ast.Node) base.QueryType {
	if query == nil {
		return base.Select
	}
	switch n := query.(type) {
	case *ast.SelectStmt:
		if hasOmniIntoClause(n) {
			return base.DDL
		}
		if containsWriteCTE(n) {
			return base.DML
		}
		return base.Select
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
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
