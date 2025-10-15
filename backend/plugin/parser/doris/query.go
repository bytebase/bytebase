package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/doris-parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_DORIS, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	// TODO: support other readonly statements like SHOW TABLES, SHOW CREATE TABLE, etc.
	result, err := ParseDorisSQL(statement)
	if err != nil {
		return false, false, err
	}
	l := &queryValidateListener{
		valid: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, result.Tree)
	if !l.valid {
		return false, false, nil
	}
	return true, true, nil
}

type queryValidateListener struct {
	*parser.BaseDorisSQLListener

	valid bool
}

func (l *queryValidateListener) EnterSingleStatement(ctx *parser.SingleStatementContext) {
	if !l.valid {
		return
	}

	stmt := ctx.Statement()
	if stmt == nil {
		return
	}

	// Allow SELECT queries and SHOW statements
	if stmt.QueryStatement() != nil {
		return
	}

	// Check for SHOW statements
	if stmt.ShowAlterStatement() != nil ||
		stmt.ShowAnalyzeStatement() != nil ||
		stmt.ShowAuthenticationStatement() != nil ||
		stmt.ShowAuthorStatement() != nil ||
		stmt.ShowBackendBlackListStatement() != nil ||
		stmt.ShowBackendsStatement() != nil ||
		stmt.ShowBackupStatement() != nil ||
		stmt.ShowBaselinePlanStatement() != nil ||
		stmt.ShowBrokerStatement() != nil ||
		stmt.ShowCatalogRecycleBinStatement() != nil ||
		stmt.ShowCatalogsStatement() != nil ||
		stmt.ShowCharsetStatement() != nil ||
		stmt.ShowClustersStatement() != nil ||
		stmt.ShowCollationStatement() != nil ||
		stmt.ShowColumnStatement() != nil ||
		stmt.ShowColumnStatsStatement() != nil ||
		stmt.ShowComputeNodesStatement() != nil ||
		stmt.ShowConvertLightSchemaChangeStatement() != nil ||
		stmt.ShowCreateDbStatement() != nil ||
		stmt.ShowCreateExternalCatalogStatement() != nil ||
		stmt.ShowCreateGroupProviderStatement() != nil ||
		stmt.ShowCreateLoadStatement() != nil ||
		stmt.ShowCreateRepositoryStatement() != nil ||
		stmt.ShowCreateRoutineLoadStatement() != nil ||
		stmt.ShowCreateSecurityIntegrationStatement() != nil ||
		stmt.ShowCreateTableStatement() != nil ||
		stmt.ShowDataCacheRulesStatement() != nil ||
		stmt.ShowDataDistributionStmt() != nil ||
		stmt.ShowDataSkewStatement() != nil ||
		stmt.ShowDataStmt() != nil ||
		stmt.ShowDataTypesStatement() != nil ||
		stmt.ShowDatabaseIdStatement() != nil ||
		stmt.ShowDatabasesStatement() != nil ||
		stmt.ShowDeleteStatement() != nil ||
		stmt.ShowDictionaryStatement() != nil ||
		stmt.ShowDynamicPartitionStatement() != nil ||
		stmt.ShowEncryptKeysStatement() != nil ||
		stmt.ShowEnginesStatement() != nil ||
		stmt.ShowEventsStatement() != nil ||
		stmt.ShowExportStatement() != nil ||
		stmt.ShowFailPointStatement() != nil ||
		stmt.ShowFrontendsDisksStatement() != nil ||
		stmt.ShowFrontendsStatement() != nil ||
		stmt.ShowFunctionsStatement() != nil ||
		stmt.ShowGrantsStatement() != nil ||
		stmt.ShowGroupProvidersStatement() != nil ||
		stmt.ShowHistogramMetaStatement() != nil ||
		stmt.ShowIndexStatement() != nil ||
		stmt.ShowJobTaskStatement() != nil ||
		stmt.ShowLastInsertStatement() != nil ||
		stmt.ShowLoadProfileStatement() != nil ||
		stmt.ShowLoadStatement() != nil ||
		stmt.ShowLoadWarningsStatement() != nil ||
		stmt.ShowMaterializedViewsStatement() != nil ||
		stmt.ShowMigrationsStatement() != nil ||
		stmt.ShowNodesStatement() != nil ||
		stmt.ShowOpenTableStatement() != nil ||
		stmt.ShowPartitionIdStatement() != nil ||
		stmt.ShowPartitionsStatement() != nil ||
		stmt.ShowPipeStatement() != nil ||
		stmt.ShowPlanAdvisorStatement() != nil ||
		stmt.ShowPluginsStatement() != nil ||
		stmt.ShowPolicyStatement() != nil ||
		stmt.ShowPrivilegesStatement() != nil ||
		stmt.ShowProcStatement() != nil ||
		stmt.ShowProcedureStatement() != nil ||
		stmt.ShowProcesslistStatement() != nil ||
		stmt.ShowProfilelistStatement() != nil ||
		stmt.ShowQueryProfileStatement() != nil ||
		stmt.ShowQueryStatsStatement() != nil ||
		stmt.ShowRepositoriesStatement() != nil ||
		stmt.ShowResourceGroupStatement() != nil ||
		stmt.ShowResourceGroupUsageStatement() != nil ||
		stmt.ShowResourceStatement() != nil ||
		stmt.ShowRestoreStatement() != nil ||
		stmt.ShowRolesStatement() != nil ||
		stmt.ShowRoutineLoadStatement() != nil ||
		stmt.ShowRoutineLoadTaskStatement() != nil ||
		stmt.ShowRunningQueriesStatement() != nil ||
		stmt.ShowSecurityIntegrationStatement() != nil ||
		stmt.ShowSmallFilesStatement() != nil ||
		stmt.ShowSnapshotStatement() != nil ||
		stmt.ShowSqlBlackListStatement() != nil ||
		stmt.ShowSqlBlockRuleStatement() != nil ||
		stmt.ShowStatsMetaStatement() != nil ||
		stmt.ShowStatusStatement() != nil ||
		stmt.ShowStorageVolumesStatement() != nil ||
		stmt.ShowStreamLoadStatement() != nil ||
		stmt.ShowSyncJobStatement() != nil ||
		stmt.ShowTableIdStatement() != nil ||
		stmt.ShowTableStatement() != nil ||
		stmt.ShowTableStatsStatement() != nil ||
		stmt.ShowTableStatusStatement() != nil ||
		stmt.ShowTabletStatement() != nil ||
		stmt.ShowTemporaryTablesStatement() != nil ||
		stmt.ShowTransactionStatement() != nil ||
		stmt.ShowTrashStatement() != nil ||
		stmt.ShowTriggersStatement() != nil ||
		stmt.ShowUserPropertyStatement() != nil ||
		stmt.ShowUserStatement() != nil ||
		stmt.ShowVariablesStatement() != nil ||
		stmt.ShowWarehousesStatement() != nil ||
		stmt.ShowWarningStatement() != nil ||
		stmt.ShowWhiteListStatement() != nil ||
		stmt.ShowWorkloadGroupsStatement() != nil {
		return
	}

	l.valid = false
}
