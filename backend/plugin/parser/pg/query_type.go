package pg

import (
	pgquery "github.com/pganalyze/pg_query_go/v6"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// no session, so we don't need to check prepare and execute.
func getQueryType(node *pgquery.Node, allSystems bool) (base.QueryType, bool) {
	switch n := node.Node.(type) {
	case *pgquery.Node_InsertStmt,
		*pgquery.Node_UpdateStmt,
		*pgquery.Node_MergeStmt,
		*pgquery.Node_CopyStmt,
		*pgquery.Node_RefreshMatViewStmt,
		*pgquery.Node_DeleteStmt:
		return base.DML, false
	case *pgquery.Node_SelectStmt:
		// SELECT INTO — define a new table from the results of a query
		if n.SelectStmt.IntoClause != nil {
			return base.DDL, false
		}

		if allSystems {
			return base.SelectInfoSchema, false
		}

		return base.Select, false
	case *pgquery.Node_VariableSetStmt:
		// Treat SAFE SET as select statement.
		return base.Select, false
	case *pgquery.Node_AlterTableStmt,
		*pgquery.Node_AlterCollationStmt,
		*pgquery.Node_AlterDomainStmt,
		*pgquery.Node_CreateStmt,
		*pgquery.Node_CreateTableSpaceStmt,
		*pgquery.Node_AlterTableSpaceOptionsStmt,
		*pgquery.Node_AlterTableMoveAllStmt,
		*pgquery.Node_CreateExtensionStmt,
		*pgquery.Node_AlterExtensionStmt,
		*pgquery.Node_AlterExtensionContentsStmt,
		*pgquery.Node_CreateFdwStmt,
		*pgquery.Node_AlterFdwStmt,
		*pgquery.Node_CreateForeignServerStmt,
		*pgquery.Node_AlterForeignServerStmt,
		*pgquery.Node_CreateForeignTableStmt,
		*pgquery.Node_CreateUserMappingStmt,
		*pgquery.Node_AlterUserMappingStmt,
		*pgquery.Node_DropUserMappingStmt,
		*pgquery.Node_ImportForeignSchemaStmt,
		*pgquery.Node_CreatePolicyStmt,
		*pgquery.Node_AlterPolicyStmt,
		*pgquery.Node_CreateAmStmt,
		*pgquery.Node_CreateTrigStmt,
		*pgquery.Node_CreateEventTrigStmt,
		*pgquery.Node_AlterEventTrigStmt,
		*pgquery.Node_CreatePlangStmt,
		*pgquery.Node_CreateRoleStmt,
		*pgquery.Node_AlterRoleStmt,
		*pgquery.Node_AlterRoleSetStmt,
		*pgquery.Node_DropRoleStmt,
		*pgquery.Node_CreateSeqStmt,
		*pgquery.Node_AlterSeqStmt,
		*pgquery.Node_CreateDomainStmt,
		*pgquery.Node_CreateOpClassStmt,
		*pgquery.Node_CreateOpFamilyStmt,
		*pgquery.Node_AlterOpFamilyStmt,
		*pgquery.Node_DropStmt,
		*pgquery.Node_TruncateStmt,
		*pgquery.Node_IndexStmt,
		*pgquery.Node_CommentStmt,
		*pgquery.Node_CreateStatsStmt,
		*pgquery.Node_AlterStatsStmt,
		*pgquery.Node_CreateFunctionStmt,
		*pgquery.Node_AlterFunctionStmt,
		*pgquery.Node_RenameStmt,
		*pgquery.Node_AlterObjectDependsStmt,
		*pgquery.Node_AlterObjectSchemaStmt,
		*pgquery.Node_AlterOwnerStmt,
		*pgquery.Node_AlterOperatorStmt,
		*pgquery.Node_AlterTypeStmt,
		*pgquery.Node_RuleStmt,
		*pgquery.Node_CompositeTypeStmt,
		*pgquery.Node_CreateEnumStmt,
		*pgquery.Node_CreateRangeStmt,
		*pgquery.Node_AlterEnumStmt,
		*pgquery.Node_ViewStmt,
		*pgquery.Node_LoadStmt,
		*pgquery.Node_CreatedbStmt,
		*pgquery.Node_AlterDatabaseStmt,
		*pgquery.Node_AlterDatabaseRefreshCollStmt,
		*pgquery.Node_AlterDatabaseSetStmt,
		*pgquery.Node_DropdbStmt,
		*pgquery.Node_AlterSystemStmt,
		*pgquery.Node_ClusterStmt,
		*pgquery.Node_VacuumStmt,
		*pgquery.Node_CreateTableAsStmt,
		*pgquery.Node_CreateConversionStmt,
		*pgquery.Node_CreateCastStmt,
		*pgquery.Node_CreateTransformStmt,
		*pgquery.Node_DropOwnedStmt,
		*pgquery.Node_ReassignOwnedStmt,
		*pgquery.Node_CreatePublicationStmt,
		*pgquery.Node_AlterPublicationStmt,
		*pgquery.Node_CreateSubscriptionStmt,
		*pgquery.Node_AlterSubscriptionStmt,
		*pgquery.Node_DropSubscriptionStmt,
		*pgquery.Node_CreateSchemaStmt:
		return base.DDL, false
	case *pgquery.Node_VariableShowStmt:
		return base.SelectInfoSchema, false
	case *pgquery.Node_ExplainStmt:
		for _, option := range n.ExplainStmt.Options {
			if defElem, ok := option.Node.(*pgquery.Node_DefElem); ok && defElem.DefElem.Defname == "analyze" {
				t, _ := getQueryType(n.ExplainStmt.Query, allSystems)
				return t, true
			}
		}

		return base.Explain, false
	}

	return base.QueryTypeUnknown, false
}
