package doris

import (
	parser "github.com/bytebase/parser/doris"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseDorisParserListener

	allSystems bool
	result     base.QueryType
}

// EnterStatementDefault is called when entering the statementDefault production (SELECT queries).
func (l *queryTypeListener) EnterStatementDefault(ctx *parser.StatementDefaultContext) {
	if ctx == nil {
		return
	}

	if ctx.Query() != nil {
		// If all tables are system tables, we should return SelectInfoSchema.
		if l.allSystems {
			l.result = base.SelectInfoSchema
		} else {
			l.result = base.Select
		}
	}
}

// EnterSupportedDmlStatementAlias is called for DML statements (INSERT, UPDATE, DELETE).
// If the statement has an EXPLAIN prefix, it's treated as a read-only query.
func (l *queryTypeListener) EnterSupportedDmlStatementAlias(ctx *parser.SupportedDmlStatementAliasContext) {
	if ctx == nil {
		return
	}

	// Check if this DML statement has an EXPLAIN prefix
	if dml := ctx.SupportedDmlStatement(); dml != nil {
		hasExplain := false
		switch stmt := dml.(type) {
		case *parser.InsertTableContext:
			hasExplain = stmt.Explain() != nil
		case *parser.UpdateContext:
			hasExplain = stmt.Explain() != nil
		case *parser.DeleteContext:
			hasExplain = stmt.Explain() != nil
		case *parser.MergeIntoContext:
			hasExplain = stmt.Explain() != nil
		}

		if hasExplain {
			// EXPLAIN on DML is read-only
			l.result = base.Select
			return
		}
	}

	l.result = base.DML
}

// EnterSupportedShowStatementAlias is called for all SHOW statements.
func (l *queryTypeListener) EnterSupportedShowStatementAlias(_ *parser.SupportedShowStatementAliasContext) {
	l.result = base.SelectInfoSchema
}

// EnterSupportedCreateStatementAlias is called for all CREATE statements.
func (l *queryTypeListener) EnterSupportedCreateStatementAlias(_ *parser.SupportedCreateStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedAlterStatementAlias is called for all ALTER statements.
func (l *queryTypeListener) EnterSupportedAlterStatementAlias(_ *parser.SupportedAlterStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedDropStatementAlias is called for all DROP statements.
func (l *queryTypeListener) EnterSupportedDropStatementAlias(_ *parser.SupportedDropStatementAliasContext) {
	l.result = base.DDL
}

// EnterMaterializedViewStatementAlias is called for materialized view statements.
func (l *queryTypeListener) EnterMaterializedViewStatementAlias(_ *parser.MaterializedViewStatementAliasContext) {
	l.result = base.DDL
}

// EnterConstraintStatementAlias is called for constraint statements.
func (l *queryTypeListener) EnterConstraintStatementAlias(_ *parser.ConstraintStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedDescribeStatementAlias is called for DESCRIBE statements (read-only).
func (l *queryTypeListener) EnterSupportedDescribeStatementAlias(_ *parser.SupportedDescribeStatementAliasContext) {
	l.result = base.SelectInfoSchema
}

// EnterSupportedUseStatementAlias is called for USE statements.
// USE statements are treated as unknown so they can be forbidden.
func (l *queryTypeListener) EnterSupportedUseStatementAlias(_ *parser.SupportedUseStatementAliasContext) {
	l.result = base.QueryTypeUnknown
}

// The following statement types default to DDL as they can modify data or schema.

// EnterSupportedLoadStatementAlias is called for LOAD statements.
func (l *queryTypeListener) EnterSupportedLoadStatementAlias(_ *parser.SupportedLoadStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedGrantRevokeStatementAlias is called for GRANT/REVOKE statements.
func (l *queryTypeListener) EnterSupportedGrantRevokeStatementAlias(_ *parser.SupportedGrantRevokeStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedAdminStatementAlias is called for ADMIN statements.
func (l *queryTypeListener) EnterSupportedAdminStatementAlias(_ *parser.SupportedAdminStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedTransactionStatementAlias is called for transaction statements.
func (l *queryTypeListener) EnterSupportedTransactionStatementAlias(_ *parser.SupportedTransactionStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedKillStatementAlias is called for KILL statements.
func (l *queryTypeListener) EnterSupportedKillStatementAlias(_ *parser.SupportedKillStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedJobStatementAlias is called for JOB statements.
func (l *queryTypeListener) EnterSupportedJobStatementAlias(_ *parser.SupportedJobStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedSetStatementAlias is called for SET statements.
func (l *queryTypeListener) EnterSupportedSetStatementAlias(_ *parser.SupportedSetStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedUnsetStatementAlias is called for UNSET statements.
func (l *queryTypeListener) EnterSupportedUnsetStatementAlias(_ *parser.SupportedUnsetStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedRefreshStatementAlias is called for REFRESH statements.
func (l *queryTypeListener) EnterSupportedRefreshStatementAlias(_ *parser.SupportedRefreshStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedCancelStatementAlias is called for CANCEL statements.
func (l *queryTypeListener) EnterSupportedCancelStatementAlias(_ *parser.SupportedCancelStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedRecoverStatementAlias is called for RECOVER statements.
func (l *queryTypeListener) EnterSupportedRecoverStatementAlias(_ *parser.SupportedRecoverStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedCleanStatementAlias is called for CLEAN statements.
func (l *queryTypeListener) EnterSupportedCleanStatementAlias(_ *parser.SupportedCleanStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedOtherStatementAlias is called for other statements not covered above.
func (l *queryTypeListener) EnterSupportedOtherStatementAlias(_ *parser.SupportedOtherStatementAliasContext) {
	l.result = base.DDL
}

// EnterSupportedStatsStatementAlias is called for stats statements.
func (l *queryTypeListener) EnterSupportedStatsStatementAlias(_ *parser.SupportedStatsStatementAliasContext) {
	l.result = base.DDL
}

// The following are SHOW statements nested within other statement categories.
// They override the parent category's DDL setting with SelectInfoSchema.

// SHOW statements in supportedLoadStatement
func (l *queryTypeListener) EnterShowCreateLoad(_ *parser.ShowCreateLoadContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowCreateRoutineLoad(_ *parser.ShowCreateRoutineLoadContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowRoutineLoad(_ *parser.ShowRoutineLoadContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowRoutineLoadTask(_ *parser.ShowRoutineLoadTaskContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowIndexAnalyzer(_ *parser.ShowIndexAnalyzerContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowIndexTokenizer(_ *parser.ShowIndexTokenizerContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowIndexTokenFilter(_ *parser.ShowIndexTokenFilterContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowIndexCharFilter(_ *parser.ShowIndexCharFilterContext) {
	l.result = base.SelectInfoSchema
}

// SHOW statements in supportedAdminStatement
func (l *queryTypeListener) EnterAdminShowReplicaDistribution(_ *parser.AdminShowReplicaDistributionContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterAdminShowReplicaStatus(_ *parser.AdminShowReplicaStatusContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterAdminShowTabletStorageFormat(_ *parser.AdminShowTabletStorageFormatContext) {
	l.result = base.SelectInfoSchema
}

// SHOW statements in supportedStatsStatement
func (l *queryTypeListener) EnterShowAnalyze(_ *parser.ShowAnalyzeContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowQueuedAnalyzeJobs(_ *parser.ShowQueuedAnalyzeJobsContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowColumnHistogramStats(_ *parser.ShowColumnHistogramStatsContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowColumnStats(_ *parser.ShowColumnStatsContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowAnalyzeTask(_ *parser.ShowAnalyzeTaskContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowIndexStats(_ *parser.ShowIndexStatsContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowTableStats(_ *parser.ShowTableStatsContext) {
	l.result = base.SelectInfoSchema
}

// SHOW statements in materializedViewStatement
func (l *queryTypeListener) EnterShowCreateMTMV(_ *parser.ShowCreateMTMVContext) {
	l.result = base.SelectInfoSchema
}

// SHOW statements in constraintStatement
func (l *queryTypeListener) EnterShowConstraint(_ *parser.ShowConstraintContext) {
	l.result = base.SelectInfoSchema
}
