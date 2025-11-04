package doris

import (
	parser "github.com/bytebase/parser/doris"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseDorisSQLListener

	allSystems bool
	result     base.QueryType
}

func (l *queryTypeListener) EnterSingleStatement(ctx *parser.SingleStatementContext) {
	if ctx == nil {
		return
	}

	s := ctx.Statement()
	if s == nil {
		return
	}

	switch {
	case s.QueryStatement() != nil:
		// If all tables are system tables, we should return SelectInfoSchema.
		if l.allSystems {
			l.result = base.SelectInfoSchema
		} else {
			l.result = base.Select
		}
	case s.InsertStatement() != nil, s.UpdateStatement() != nil, s.DeleteStatement() != nil:
		l.result = base.DML
	case s.ShowAlterStatement() != nil,
		s.ShowAnalyzeStatement() != nil,
		s.ShowAuthenticationStatement() != nil,
		s.ShowAuthorStatement() != nil,
		s.ShowBackendBlackListStatement() != nil,
		s.ShowBackendsStatement() != nil,
		s.ShowBackupStatement() != nil,
		s.ShowBaselinePlanStatement() != nil,
		s.ShowBrokerStatement() != nil,
		s.ShowCatalogRecycleBinStatement() != nil,
		s.ShowCatalogsStatement() != nil,
		s.ShowCharsetStatement() != nil,
		s.ShowClustersStatement() != nil,
		s.ShowCollationStatement() != nil,
		s.ShowColumnStatement() != nil,
		s.ShowColumnStatsStatement() != nil,
		s.ShowComputeNodesStatement() != nil,
		s.ShowConvertLightSchemaChangeStatement() != nil,
		s.ShowCreateDbStatement() != nil,
		s.ShowCreateExternalCatalogStatement() != nil,
		s.ShowCreateGroupProviderStatement() != nil,
		s.ShowCreateLoadStatement() != nil,
		s.ShowCreateRepositoryStatement() != nil,
		s.ShowCreateRoutineLoadStatement() != nil,
		s.ShowCreateSecurityIntegrationStatement() != nil,
		s.ShowCreateTableStatement() != nil,
		s.ShowDataCacheRulesStatement() != nil,
		s.ShowDataDistributionStmt() != nil,
		s.ShowDataSkewStatement() != nil,
		s.ShowDataStmt() != nil,
		s.ShowDataTypesStatement() != nil,
		s.ShowDatabaseIdStatement() != nil,
		s.ShowDatabasesStatement() != nil,
		s.ShowDeleteStatement() != nil,
		s.ShowDictionaryStatement() != nil,
		s.ShowDynamicPartitionStatement() != nil,
		s.ShowEncryptKeysStatement() != nil,
		s.ShowEnginesStatement() != nil,
		s.ShowEventsStatement() != nil,
		s.ShowExportStatement() != nil,
		s.ShowFailPointStatement() != nil,
		s.ShowFrontendsDisksStatement() != nil,
		s.ShowFrontendsStatement() != nil,
		s.ShowFunctionsStatement() != nil,
		s.ShowGrantsStatement() != nil,
		s.ShowGroupProvidersStatement() != nil,
		s.ShowHistogramMetaStatement() != nil,
		s.ShowIndexStatement() != nil,
		s.ShowJobTaskStatement() != nil,
		s.ShowLastInsertStatement() != nil,
		s.ShowLoadProfileStatement() != nil,
		s.ShowLoadStatement() != nil,
		s.ShowLoadWarningsStatement() != nil,
		s.ShowMaterializedViewsStatement() != nil,
		s.ShowMigrationsStatement() != nil,
		s.ShowNodesStatement() != nil,
		s.ShowOpenTableStatement() != nil,
		s.ShowPartitionIdStatement() != nil,
		s.ShowPartitionsStatement() != nil,
		s.ShowPipeStatement() != nil,
		s.ShowPlanAdvisorStatement() != nil,
		s.ShowPluginsStatement() != nil,
		s.ShowPolicyStatement() != nil,
		s.ShowPrivilegesStatement() != nil,
		s.ShowProcStatement() != nil,
		s.ShowProcedureStatement() != nil,
		s.ShowProcesslistStatement() != nil,
		s.ShowProfilelistStatement() != nil,
		s.ShowQueryProfileStatement() != nil,
		s.ShowQueryStatsStatement() != nil,
		s.ShowRepositoriesStatement() != nil,
		s.ShowResourceGroupStatement() != nil,
		s.ShowResourceGroupUsageStatement() != nil,
		s.ShowResourceStatement() != nil,
		s.ShowRestoreStatement() != nil,
		s.ShowRolesStatement() != nil,
		s.ShowRoutineLoadStatement() != nil,
		s.ShowRoutineLoadTaskStatement() != nil,
		s.ShowRunningQueriesStatement() != nil,
		s.ShowSecurityIntegrationStatement() != nil,
		s.ShowSmallFilesStatement() != nil,
		s.ShowSnapshotStatement() != nil,
		s.ShowSqlBlackListStatement() != nil,
		s.ShowSqlBlockRuleStatement() != nil,
		s.ShowStatsMetaStatement() != nil,
		s.ShowStatusStatement() != nil,
		s.ShowStorageVolumesStatement() != nil,
		s.ShowStreamLoadStatement() != nil,
		s.ShowSyncJobStatement() != nil,
		s.ShowTableIdStatement() != nil,
		s.ShowTableStatement() != nil,
		s.ShowTableStatsStatement() != nil,
		s.ShowTableStatusStatement() != nil,
		s.ShowTabletStatement() != nil,
		s.ShowTemporaryTablesStatement() != nil,
		s.ShowTransactionStatement() != nil,
		s.ShowTrashStatement() != nil,
		s.ShowTriggersStatement() != nil,
		s.ShowUserPropertyStatement() != nil,
		s.ShowUserStatement() != nil,
		s.ShowVariablesStatement() != nil,
		s.ShowWarehousesStatement() != nil,
		s.ShowWarningStatement() != nil,
		s.ShowWhiteListStatement() != nil,
		s.ShowWorkloadGroupsStatement() != nil:
		l.result = base.SelectInfoSchema
	default:
		l.result = base.DDL
	}
}
