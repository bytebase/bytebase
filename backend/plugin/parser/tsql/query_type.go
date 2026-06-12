package tsql

import (
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// classifyQueryType classifies an omni AST node into a QueryType.
// allSystems indicates whether all referenced tables are system/info_schema tables.
func classifyQueryType(node ast.Node, allSystems bool) base.QueryType {
	if node == nil {
		return base.QueryTypeUnknown
	}

	switch n := node.(type) {
	case *ast.SelectStmt:
		// SELECT ... INTO new_table creates a table — it is a write (DDL), not a read.
		// Without this the write would take the SELECT path and execute under read access.
		// INTO may sit on the first arm of a set operation, not the root. An INTO #temp
		// (or ##temp) target stays a read: it materialises a session-scoped tempdb table
		// that the extractor registers in TempTables for follow-up statements.
		if target := selectIntoTarget(n); target != nil && !strings.HasPrefix(target.Object, "#") {
			return base.DDL
		}
		if allSystems {
			return base.SelectInfoSchema
		}
		return base.Select

	// Safe read-only statements treated as Select (parity with the pre-omni
	// Another_statement → SET/SETUSER/DECLARE branch).
	case *ast.SetStmt, *ast.SetOptionStmt, *ast.DeclareStmt:
		return base.Select

	// DML statements.
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt,
		*ast.BulkInsertStmt, *ast.InsertBulkStmt, *ast.CopyIntoStmt,
		*ast.ReadtextStmt, *ast.WritetextStmt, *ast.UpdatetextStmt,
		*ast.ExecStmt, *ast.ReceiveStmt, *ast.PredictStmt:
		return base.DML

	// DDL statements.
	case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropStmt,
		*ast.TruncateStmt, *ast.RenameStmt,
		*ast.CreateIndexStmt, *ast.AlterIndexStmt,
		*ast.CreateViewStmt,
		*ast.CreateTriggerStmt, *ast.EnableDisableTriggerStmt,
		*ast.CreateFunctionStmt, *ast.CreateProcedureStmt,
		*ast.CreateDatabaseStmt, *ast.AlterDatabaseStmt,
		*ast.CreateSchemaStmt, *ast.AlterSchemaStmt,
		*ast.CreateTypeStmt,
		*ast.CreateSequenceStmt, *ast.AlterSequenceStmt,
		*ast.CreateSynonymStmt,
		*ast.GrantStmt, *ast.SecurityStmt, *ast.SecurityKeyStmt, *ast.SecurityPolicyStmt,
		*ast.SensitivityClassificationStmt, *ast.SignatureStmt,
		*ast.CreateStatisticsStmt, *ast.UpdateStatisticsStmt, *ast.DropStatisticsStmt,
		*ast.CreatePartitionFunctionStmt, *ast.AlterPartitionFunctionStmt,
		*ast.CreatePartitionSchemeStmt, *ast.AlterPartitionSchemeStmt,
		*ast.CreateFulltextIndexStmt, *ast.AlterFulltextIndexStmt,
		*ast.CreateFulltextCatalogStmt, *ast.AlterFulltextCatalogStmt,
		*ast.CreateFulltextStoplistStmt, *ast.AlterFulltextStoplistStmt, *ast.DropFulltextStoplistStmt,
		*ast.CreateSearchPropertyListStmt, *ast.AlterSearchPropertyListStmt, *ast.DropSearchPropertyListStmt,
		*ast.CreateXmlSchemaCollectionStmt, *ast.AlterXmlSchemaCollectionStmt,
		*ast.CreateXmlIndexStmt, *ast.CreateSelectiveXmlIndexStmt,
		*ast.CreateSpatialIndexStmt, *ast.CreateJsonIndexStmt, *ast.CreateVectorIndexStmt,
		*ast.CreateAggregateStmt, *ast.DropAggregateStmt,
		*ast.CreateAssemblyStmt, *ast.AlterAssemblyStmt,
		*ast.CreateMaterializedViewStmt, *ast.AlterMaterializedViewStmt,
		*ast.CreateExternalTableAsSelectStmt, *ast.CreateTableCloneStmt,
		*ast.CreateTableAsSelectStmt, *ast.CreateRemoteTableAsSelectStmt,
		*ast.CreateFederationStmt, *ast.AlterFederationStmt, *ast.DropFederationStmt, *ast.UseFederationStmt,
		*ast.AlterServerConfigurationStmt:
		return base.DDL

	default:
		// Flow control (IF/WHILE/BEGIN-END/TRY-CATCH/RETURN/BREAK/CONTINUE/GOTO/LABEL/WAITFOR),
		// PRINT/RAISERROR/THROW, DBCC, BACKUP/RESTORE, transactions, cursors, and other
		// statements we do not classify — report as Unknown to match the pre-omni behavior.
		return base.QueryTypeUnknown
	}
}
