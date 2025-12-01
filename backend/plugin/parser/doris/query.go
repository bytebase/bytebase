package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/doris"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_DORIS, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	// TODO: support other readonly statements like SHOW TABLES, SHOW CREATE TABLE, etc.
	results, err := ParseDorisSQL(statement)
	if err != nil {
		return false, false, err
	}
	for _, result := range results {
		l := &queryValidateListener{
			valid: true,
		}
		antlr.ParseTreeWalkerDefault.Walk(l, result.Tree)
		if !l.valid {
			return false, false, nil
		}
	}
	return true, true, nil
}

type queryValidateListener struct {
	*parser.BaseDorisParserListener

	valid bool
}

// EnterStatementDefault is called when entering the statementDefault production (SELECT queries).
// SELECT statements are valid queries.
func (l *queryValidateListener) EnterStatementDefault(ctx *parser.StatementDefaultContext) {
	if !l.valid {
		return
	}
	// SELECT queries are allowed
	if ctx != nil && ctx.Query() != nil {
		return
	}
}

// EnterSupportedShowStatementAlias is called for all SHOW statements.
// SHOW statements are valid read-only queries.
func (*queryValidateListener) EnterSupportedShowStatementAlias(_ *parser.SupportedShowStatementAliasContext) {
	// SHOW statements are allowed
}

// EnterSupportedDmlStatementAlias is called for DML statements (INSERT, UPDATE, DELETE).
// DML statements are NOT valid read-only queries, unless they have an EXPLAIN prefix.
func (l *queryValidateListener) EnterSupportedDmlStatementAlias(ctx *parser.SupportedDmlStatementAliasContext) {
	if ctx == nil {
		return
	}

	// Check if this DML statement has an EXPLAIN prefix (read-only)
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
			// EXPLAIN on DML is read-only, so it's valid
			return
		}
	}

	l.valid = false
}

// EnterSupportedCreateStatementAlias is called for CREATE statements.
func (l *queryValidateListener) EnterSupportedCreateStatementAlias(_ *parser.SupportedCreateStatementAliasContext) {
	l.valid = false
}

// EnterSupportedAlterStatementAlias is called for ALTER statements.
func (l *queryValidateListener) EnterSupportedAlterStatementAlias(_ *parser.SupportedAlterStatementAliasContext) {
	l.valid = false
}

// EnterSupportedDropStatementAlias is called for DROP statements.
func (l *queryValidateListener) EnterSupportedDropStatementAlias(_ *parser.SupportedDropStatementAliasContext) {
	l.valid = false
}

// EnterMaterializedViewStatementAlias is called for materialized view statements.
func (l *queryValidateListener) EnterMaterializedViewStatementAlias(_ *parser.MaterializedViewStatementAliasContext) {
	l.valid = false
}

// EnterConstraintStatementAlias is called for constraint statements.
func (l *queryValidateListener) EnterConstraintStatementAlias(_ *parser.ConstraintStatementAliasContext) {
	l.valid = false
}

// EnterSupportedLoadStatementAlias is called for LOAD statements.
func (l *queryValidateListener) EnterSupportedLoadStatementAlias(_ *parser.SupportedLoadStatementAliasContext) {
	l.valid = false
}

// EnterSupportedGrantRevokeStatementAlias is called for GRANT/REVOKE statements.
func (l *queryValidateListener) EnterSupportedGrantRevokeStatementAlias(_ *parser.SupportedGrantRevokeStatementAliasContext) {
	l.valid = false
}

// EnterSupportedAdminStatementAlias is called for ADMIN statements.
func (l *queryValidateListener) EnterSupportedAdminStatementAlias(_ *parser.SupportedAdminStatementAliasContext) {
	l.valid = false
}

// EnterSupportedTransactionStatementAlias is called for transaction statements.
func (l *queryValidateListener) EnterSupportedTransactionStatementAlias(_ *parser.SupportedTransactionStatementAliasContext) {
	l.valid = false
}

// EnterSupportedKillStatementAlias is called for KILL statements.
func (l *queryValidateListener) EnterSupportedKillStatementAlias(_ *parser.SupportedKillStatementAliasContext) {
	l.valid = false
}

// EnterSupportedJobStatementAlias is called for JOB statements.
func (l *queryValidateListener) EnterSupportedJobStatementAlias(_ *parser.SupportedJobStatementAliasContext) {
	l.valid = false
}

// EnterSupportedSetStatementAlias is called for SET statements.
func (l *queryValidateListener) EnterSupportedSetStatementAlias(_ *parser.SupportedSetStatementAliasContext) {
	l.valid = false
}

// EnterSupportedUnsetStatementAlias is called for UNSET statements.
func (l *queryValidateListener) EnterSupportedUnsetStatementAlias(_ *parser.SupportedUnsetStatementAliasContext) {
	l.valid = false
}

// EnterSupportedRefreshStatementAlias is called for REFRESH statements.
func (l *queryValidateListener) EnterSupportedRefreshStatementAlias(_ *parser.SupportedRefreshStatementAliasContext) {
	l.valid = false
}

// EnterSupportedCancelStatementAlias is called for CANCEL statements.
func (l *queryValidateListener) EnterSupportedCancelStatementAlias(_ *parser.SupportedCancelStatementAliasContext) {
	l.valid = false
}

// EnterSupportedRecoverStatementAlias is called for RECOVER statements.
func (l *queryValidateListener) EnterSupportedRecoverStatementAlias(_ *parser.SupportedRecoverStatementAliasContext) {
	l.valid = false
}

// EnterSupportedCleanStatementAlias is called for CLEAN statements.
func (l *queryValidateListener) EnterSupportedCleanStatementAlias(_ *parser.SupportedCleanStatementAliasContext) {
	l.valid = false
}

// EnterSupportedOtherStatementAlias is called for other unsupported statements.
func (l *queryValidateListener) EnterSupportedOtherStatementAlias(_ *parser.SupportedOtherStatementAliasContext) {
	l.valid = false
}

// EnterSupportedStatsStatementAlias is called for stats statements.
func (l *queryValidateListener) EnterSupportedStatsStatementAlias(_ *parser.SupportedStatsStatementAliasContext) {
	l.valid = false
}
