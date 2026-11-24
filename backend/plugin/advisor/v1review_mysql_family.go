package advisor

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// MySQLFamilyOptions configures a MySQL-family rule builder for one engine.
type MySQLFamilyOptions[N any] struct {
	Extract        func(Context) ([]V1Stmt[N], error)
	ByteOffsetLine func(sql string, byteOffset int) int32
	// TiDBQuirks preserves V1 position, text-trimming, and message differences
	// between the TiDB adapter lineage and the MySQL/MariaDB/OceanBase lineage.
	TiDBQuirks bool

	// Optional AST node predicates for V1 panic parity and position selection.
	DefinesColumn       func(N) bool
	ReadsColumnType     func(N) bool
	DropsColumn         func(N) bool
	IsCreateTable       func(N) bool
	IsAlterTable        func(N) bool
	CreateTableLocStart func(N) (int, bool)
}

// MySQLFamilyRuleSet declares the omni v1review rule implementations for one
// MySQL-family engine. Nil fields are omitted from registration.
type MySQLFamilyRuleSet[N any] struct {
	OnlineMigration            func([]N, v1review.Schema, string, int64) []v1review.Finding
	MaxStatementsInTransaction bool
	InsertRowLimitProbes       func([]N) []int
	InsertRowLimit             func([]N, map[int]v1review.Probe, int) []v1review.Finding
	AffectedRowLimitProbes     func([]N) []int
	AffectedRowLimit           func([]N, map[int]v1review.Probe, int) []v1review.Finding
	DMLDryRunProbes            func([]N) []int
	DMLDryRun                  func([]N, map[int]v1review.Probe) []v1review.Finding

	ColumnAutoIncrementInitialValue           func([]N, int) []v1review.Finding
	ColumnAutoIncrementMustInteger            func([]N) []v1review.Finding
	ColumnAutoIncrementMustUnsigned           func([]N) []v1review.Finding
	ColumnComment                             func([]N, bool, int) []v1review.Finding
	ColumnCurrentTimeCountLimit               func([]N) []v1review.Finding
	ColumnDisallowChange                      func([]N) []v1review.Finding
	ColumnDisallowChangeType                  func([]N, v1review.Schema) []v1review.Finding
	ColumnDisallowChangingOrder               func([]N) []v1review.Finding
	ColumnDisallowDrop                        func([]N) []v1review.Finding
	ColumnDisallowDropInIndex                 func([]N, v1review.Schema) []v1review.Finding
	ColumnDisallowSetCharset                  func([]N) []v1review.Finding
	ColumnMaximumCharacterLength              func([]N, int) []v1review.Finding
	ColumnMaximumVarcharLength                func([]N, int) []v1review.Finding
	ColumnNoNull                              func([]N, v1review.Schema) []v1review.Finding
	ColumnRequireCharset                      func([]N) []v1review.Finding
	ColumnRequireCollation                    func([]N) []v1review.Finding
	ColumnRequireDefault                      func([]N) []v1review.Finding
	ColumnRequired                            func([]N, []string) []v1review.Finding
	ColumnSetDefaultForNotNull                func([]N) []v1review.Finding
	ColumnTypeDisallowList                    func([]N, []string) []v1review.Finding
	DatabaseDropEmptyDatabase                 func([]N, v1review.Schema) []v1review.Finding
	EngineMySQLUseInnoDB                      func([]N) []v1review.Finding
	IndexKeyNumberLimit                       func([]N, int) []v1review.Finding
	IndexNoDuplicateColumn                    func([]N) []v1review.Finding
	IndexPKTypeLimit                          func([]N, v1review.Schema) ([]v1review.Finding, error)
	IndexPrimaryKeyTypeAllowlist              func([]N, []string, v1review.Schema) ([]v1review.Finding, error)
	IndexTotalNumberLimit                     func([]N, int, v1review.Schema) ([]v1review.Finding, error)
	IndexTypeAllowList                        func([]N, []string) []v1review.Finding
	IndexTypeNoBlob                           func([]N, v1review.Schema) ([]v1review.Finding, error)
	NamingAutoIncrementColumn                 func([]N, v1review.NamingConfig) ([]v1review.Finding, error)
	NamingColumn                              func([]N, v1review.NamingConfig) ([]v1review.Finding, error)
	NamingForeignKey                          func([]N, v1review.TemplateConfig) ([]v1review.Finding, error)
	NamingIdentifierNoKeyword                 func([]N) []v1review.Finding
	NamingIndex                               func([]N, v1review.TemplateConfig, v1review.Schema) []v1review.Finding
	NamingTable                               func([]N, v1review.NamingConfig) ([]v1review.Finding, error)
	NamingUniqueKey                           func([]N, v1review.TemplateConfig, v1review.Schema) []v1review.Finding
	SchemaBackwardCompatibility               func([]N) []v1review.Finding
	StatementAddColumnWithoutPosition         func([]N) []v1review.Finding
	StatementDisallowCommit                   func([]N) []v1review.Finding
	StatementDisallowLimit                    func([]N) []v1review.Finding
	StatementDisallowOrderBy                  func([]N) []v1review.Finding
	StatementDisallowTruncate                 func([]N) []v1review.Finding
	StatementInsertDisallowOrderByRand        func([]N) []v1review.Finding
	StatementInsertMustSpecifyColumn          func([]N) []v1review.Finding
	StatementMaxExecutionTime                 func([]N) []v1review.Finding
	StatementMergeAlterTable                  func([]N) []v1review.Finding
	StatementRequireAlgorithmOption           func([]N) []v1review.Finding
	StatementRequireLockOption                func([]N) []v1review.Finding
	StatementSelectNoSelectAll                func([]N) []v1review.Finding
	StatementWhereDisallowFunctionsAndCalc    func([]N, v1review.Schema) []v1review.Finding
	StatementWhereMaximumLogicalOperatorCount func([]N, int) []v1review.Finding
	StatementWhereNoEqualNull                 func([]N) []v1review.Finding
	StatementWhereNoLeadingWildcardLike       func([]N) []v1review.Finding
	StatementWhereRequireSelect               func([]N) []v1review.Finding
	StatementWhereRequireUpdateDelete         func([]N) []v1review.Finding
	SystemCharsetAllowlist                    func([]N, []string) []v1review.Finding
	SystemCollationAllowlist                  func([]N, []string) []v1review.Finding
	SystemEventDisallowCreate                 func([]N) []v1review.Finding
	SystemFunctionDisallowCreate              func([]N) []v1review.Finding
	SystemFunctionDisallowedList              func([]N, []string) []v1review.Finding
	SystemProcedureDisallowCreate             func([]N) []v1review.Finding
	SystemViewDisallowCreate                  func([]N) []v1review.Finding
	TableComment                              func([]N, *v1review.TableCommentConfig) []v1review.Finding
	TableDisallowDDL                          func([]N, []string) []v1review.Finding
	TableDisallowDML                          func([]N, []string) []v1review.Finding
	TableDisallowPartition                    func([]N) []v1review.Finding
	TableDisallowSetCharset                   func([]N) []v1review.Finding
	TableDisallowTrigger                      func([]N) []v1review.Finding
	TableDropNaming                           func([]N, v1review.NamingConfig) ([]v1review.Finding, error)
	TableLimitSize                            func([]N, *metadatapb.DatabaseSchemaMetadata, int) []v1review.Finding
	TableNoDuplicateIndex                     func([]N) []v1review.Finding
	TableNoForeignKey                         func([]N) []v1review.Finding
	TableRequireCharset                       func([]N) []v1review.Finding
	TableRequireCollation                     func([]N) []v1review.Finding
	TableRequirePK                            func([]N, *v1review.Schema) []v1review.Finding
	TableTextFieldsTotalLength                func([]N, *v1review.Schema, int) []v1review.Finding
}

// RegisterMySQLFamily registers BUILTIN_WALK_THROUGH_CHECK and every non-nil rule
// in rules for engine, returning the underlying V1Engine for any engine-specific rules.
func RegisterMySQLFamily[N any](engine storepb.Engine, opt MySQLFamilyOptions[N], rules MySQLFamilyRuleSet[N]) *V1Engine[N] {
	e := newV1EngineWithOptions(v1EngineOptions[N]{
		Extract:             opt.Extract,
		ByteOffsetLine:      opt.ByteOffsetLine,
		PosMode:             V1PosContentStart,
		TiDBQuirks:          opt.TiDBQuirks,
		IsAlterTable:        opt.IsAlterTable,
		CreateTableLocStart: opt.CreateTableLocStart,
	})
	Register(engine, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &BuiltinWalkThroughCheckAdvisor{})
	for ruleType, fn := range buildMySQLFamilyAdvisors(engine, e, opt, rules) {
		Register(engine, ruleType, fn)
	}
	return e
}

var (
	dmlLimitCodes   = map[string]code.Code{"INSERT": code.InsertUseLimit, "UPDATE": code.UpdateUseLimit, "DELETE": code.DeleteUseLimit}
	dmlOrderByCodes = map[string]code.Code{"UPDATE": code.UpdateUseOrderBy, "DELETE": code.DeleteUseOrderBy}
)

func buildMySQLFamilyAdvisors[N any](
	engine storepb.Engine,
	e *V1Engine[N],
	opt MySQLFamilyOptions[N],
	rules MySQLFamilyRuleSet[N],
) map[storepb.SQLReviewRule_Type]Func {
	advisors := make(map[storepb.SQLReviewRule_Type]Func)

	addSimple := func(ruleType storepb.SQLReviewRule_Type, fn func([]N) []v1review.Finding, c code.Code, pos V1PosFunc[N], content func(V1Stmt[N], v1review.Finding) string) {
		if fn != nil {
			advisors[ruleType] = e.SimpleStmt(fn, c, content, pos)
		}
	}
	fmtStmt := func(format string) func(V1Stmt[N], v1review.Finding) string {
		return func(stmt V1Stmt[N], _ v1review.Finding) string {
			return fmt.Sprintf(format, e.stmtText(stmt, false))
		}
	}
	addDisallowCreate := func(ruleType storepb.SQLReviewRule_Type, fn func([]N) []v1review.Finding, c code.Code, object string) {
		addSimple(ruleType, fn, c, e.posBaseLineTwice, fmtStmt(object+" is forbidden, but \"%s\" creates"))
	}
	addNumber := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, int) []v1review.Finding, missingMsg string, positiveOnly bool, c code.Code, pos V1PosFunc[N], content func(V1Stmt[N], v1review.Finding, int) string) {
		if fn != nil {
			advisors[ruleType] = e.numberStmtWithMsg(fn, missingMsg, positiveOnly, c, content, pos)
		}
	}
	addStringArray := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, []string) []v1review.Finding, c code.Code, pos V1PosFunc[N], content func(V1Stmt[N], v1review.Finding) string) {
		if fn != nil {
			advisors[ruleType] = e.StringArrayStmt(fn, "", c, "", content, pos)
		}
	}
	addNaming := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error), c code.Code, missingMsg, kind string, checkLength bool, subject func(v1review.Finding) string, pos V1PosFunc[N]) {
		if fn != nil {
			advisors[ruleType] = e.NamingRule(fn, c, missingMsg, kind, checkLength, subject, pos)
		}
	}
	addSimpleBatch := func(ruleType storepb.SQLReviewRule_Type, fn func([]N) []v1review.Finding, c code.Code, content func(v1review.Finding) string, pos V1PosFunc[N]) {
		if fn != nil {
			advisors[ruleType] = e.SimpleBatchNodes(fn, c, "", content, pos)
		}
	}
	addOriginalSchemaBatch := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, v1review.Schema) ([]v1review.Finding, error), c code.Code, content func(v1review.Finding) string) {
		if fn != nil {
			advisors[ruleType] = e.SchemaBatch(c, func(_ Context, f v1review.Finding) string { return content(f) }, func(checkCtx Context, nodes []N) ([]v1review.Finding, *model.DatabaseMetadata, error) {
				findings, err := fn(nodes, V1SchemaOf(checkCtx.OriginalMetadata))
				return findings, checkCtx.OriginalMetadata, err
			}, e.PosFinding)
		}
	}

	// Simple per-statement rules.
	addSimple(storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, rules.ColumnAutoIncrementMustInteger, code.AutoIncrementColumnNotInteger, e.posAbsOrFinding, e.FmtTableCol("Auto-increment column `%s`.`%s` requires integer type"))
	addSimple(storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, rules.ColumnAutoIncrementMustUnsigned, code.AutoIncrementColumnSigned, e.posAbsOrFinding, e.FmtTableCol("Auto-increment column `%s`.`%s` is not UNSIGNED type"))
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, rules.ColumnDisallowChange, code.UseChangeColumnStatement, e.posStatementAbsOrStmt, fmtStmt("\"%s\" contains CHANGE COLUMN statement"))
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, rules.ColumnDisallowChangingOrder, code.ChangeColumnOrder, e.posStatementAbsOrStmt, fmtStmt("\"%s\" changes column order"))
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_DROP, rules.ColumnDisallowDrop, code.DropColumn, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("drops column \"%s\" of table \"%s\"", f.Name, f.Table)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, rules.ColumnDisallowSetCharset, code.SetColumnCharset, e.PosStatement, fmtStmt("Disallow set column charset but \"%s\" does"))
	addSimple(storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET, rules.ColumnRequireCharset, code.NoCharset, e.posAbsOrFinding, e.FmtName("Column %s does not have a character set specified"))
	addSimple(storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION, rules.ColumnRequireCollation, code.NoCollation, e.posAbsOrFinding, e.FmtName("Column %s does not have a collation specified"))
	addSimple(storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, rules.ColumnRequireDefault, code.NoDefault, e.posAbsOrFinding, e.FmtTableCol("Column `%s`.`%s` doesn't have DEFAULT."))
	addSimple(storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, rules.ColumnSetDefaultForNotNull, code.NotNullColumnWithNoDefault, e.posAlterFirstToken, e.FmtTableCol("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT"))
	addSimple(storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, rules.EngineMySQLUseInnoDB, code.NotInnoDBEngine, e.PosSearchOrStart, e.FmtStmt("\"%s;\" doesn't use InnoDB engine"))
	addSimple(storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, rules.IndexNoDuplicateColumn, code.DuplicateColumnInIndex, e.PosFinding, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("%s `%s` has duplicate column `%s`.`%s`", f.Value, keyName(stmt.Text, f, e.tidbQuirks), f.Table, f.Other)
	})
	addSimple(storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, rules.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier, e.posSearchOrFinding, e.FmtName("Identifier %q is a keyword and should be avoided"))
	addSimple(storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION, rules.StatementAddColumnWithoutPosition, code.StatementAddColumnWithPosition, e.PosFinding, e.FmtName("add column with position \"%s\""))
	addSimple(storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, rules.StatementDisallowCommit, code.StatementDisallowCommit, e.posFindingOrTiDBFirstToken, fmtStmt("Commit is not allowed, related statement: \"%s\""))
	addSimple(storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, rules.StatementInsertDisallowOrderByRand, code.InsertUseOrderByRand, e.posFindingOrTiDBFirstToken, fmtStmt("\"%s\" uses ORDER BY RAND in the INSERT statement"))
	addSimple(storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, rules.StatementInsertMustSpecifyColumn, code.InsertNotSpecifyColumn, e.posAbsOrFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", e.stmtText(stmt, true))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, rules.StatementRequireAlgorithmOption, code.StatementNoAlgorithmOption, e.PosFinding, e.FmtName("ALTER TABLE statement should include %s option"))
	addSimple(storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, rules.StatementRequireLockOption, code.StatementNoLockOption, e.PosFinding, e.FmtName("ALTER TABLE statement should include %s option"))
	addSimple(storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, rules.StatementSelectNoSelectAll, code.StatementSelectAll, e.posFindingOrTiDBFirstToken, fmtStmt("\"%s\" uses SELECT all"))
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL, rules.StatementWhereNoEqualNull, code.StatementWhereNoEqualNull, e.PosFinding, fmtStmt("WHERE clause contains equal null: %s"))
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, rules.StatementWhereNoLeadingWildcardLike, code.StatementLeadingWildcardLike, e.posFindingOrTiDBFirstToken, fmtStmt("\"%s\" uses leading wildcard LIKE"))
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, rules.StatementWhereRequireSelect, code.StatementNoWhere, e.posFindingOrTiDBFirstToken, fmtStmt("\"%s\" requires WHERE clause"))
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, rules.StatementWhereRequireUpdateDelete, code.StatementNoWhere, e.posFindingOrTiDBFirstToken, fmtStmt("\"%s\" requires WHERE clause"))
	addSimple(storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, rules.TableDisallowPartition, code.CreateTablePartition, e.PosStatement, fmtStmt("Table partition is forbidden, but \"%s\" creates"))
	addSimple(storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET, rules.TableDisallowSetCharset, code.DisallowSetCharset, e.PosStatement, fmtStmt("Set charset on tables is disallowed, but \"%s\" uses"))
	addSimple(storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET, rules.TableRequireCharset, code.NoCharset, e.PosStatement, e.FmtName("Table %s does not have a character set specified"))
	addSimple(storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION, rules.TableRequireCollation, code.NoCollation, e.PosStatement, e.FmtName("Table %s does not have a collation specified"))

	// Disallow create rules.
	addDisallowCreate(storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE, rules.SystemEventDisallowCreate, code.DisallowCreateEvent, "Event")
	addDisallowCreate(storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE, rules.SystemFunctionDisallowCreate, code.DisallowCreateFunction, "Function")
	addDisallowCreate(storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, rules.SystemProcedureDisallowCreate, code.DisallowCreateProcedure, "Procedure")
	addDisallowCreate(storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE, rules.SystemViewDisallowCreate, code.DisallowCreateView, "View")
	addDisallowCreate(storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER, rules.TableDisallowTrigger, code.CreateTableTrigger, "Trigger")

	// Number-payload per-statement rules.
	addNumber(storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, rules.ColumnAutoIncrementInitialValue, "number_payload is required for column auto increment initial value rule", false, code.AutoIncrementColumnInitialValueNotMatch, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding, value int) string {
		return fmt.Sprintf("The initial auto-increment value in table `%s` is %s, which doesn't equal %v", f.Table, f.Value, value)
	})
	addNumber(storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, rules.ColumnMaximumCharacterLength, "number_payload is required for column maximum character length rule", true, code.CharLengthExceedsLimit, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding, maximum int) string {
		if e.tidbQuirks {
			return fmt.Sprintf("The length of the CHAR column `%s` is %s, bigger than %d, please use VARCHAR instead", f.Name, f.Value, maximum)
		}
		return fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", f.Table, f.Name, maximum)
	})
	addNumber(storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, rules.ColumnMaximumVarcharLength, "number_payload is required for column maximum varchar length rule", true, code.VarcharLengthExceedsLimit, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding, maximum int) string {
		return fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", f.Table, f.Name, maximum)
	})
	addNumber(storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, rules.IndexKeyNumberLimit, "", false, code.IndexKeyNumberExceedsLimit, e.PosFinding, func(stmt V1Stmt[N], f v1review.Finding, limit int) string {
		return fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", keyName(stmt.Text, f, e.tidbQuirks), f.Table, limit)
	})
	addNumber(storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT, rules.StatementWhereMaximumLogicalOperatorCount, "", false, code.StatementWhereMaximumLogicalOperatorCount, e.PosFinding, func(stmt V1Stmt[N], f v1review.Finding, maximum int) string {
		op := "the OR predicate"
		if f.Name == "IN" {
			op = "IN predicate"
		}
		return fmt.Sprintf("Number of tokens (%s) in %s operation exceeds limit (%d) in statement %q.", f.Value, op, maximum, strings.TrimSpace(stmt.Text))
	})

	// String-array per-statement rules.
	addStringArray(storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, rules.ColumnTypeDisallowList, code.DisabledColumnType, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Disallow column type %s but column `%s`.`%s` is", f.Value, f.Table, f.Name)
	})
	addStringArray(storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST, rules.IndexTypeAllowList, code.IndexTypeNotAllowed, e.PosFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Index type `%s` is not allowed", f.Value)
	})
	addStringArray(storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, rules.SystemCharsetAllowlist, code.DisabledCharset, e.posAllowlist, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("\"%s\" used disabled charset '%s'", strings.TrimSpace(stmt.Text), f.Name)
	})
	addStringArray(storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, rules.SystemCollationAllowlist, code.DisabledCollation, e.posAllowlist, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("\"%s\" used disabled collation '%s'", strings.TrimSpace(stmt.Text), f.Name)
	})
	addStringArray(storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST, rules.SystemFunctionDisallowedList, code.DisabledFunction, e.posBaseLineTwice, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", f.Name, strings.TrimSpace(stmt.Text))
	})
	addStringArray(storepb.SQLReviewRule_TABLE_DISALLOW_DDL, rules.TableDisallowDDL, code.TableDisallowDDL, e.PosFinding, e.FmtName("DDL is disallowed on table %s."))
	addStringArray(storepb.SQLReviewRule_TABLE_DISALLOW_DML, rules.TableDisallowDML, code.TableDisallowDML, e.PosFinding, e.FmtName("DML is disallowed on table %s."))

	// Regex naming rules.
	btTableCol := func(f v1review.Finding) string { return fmt.Sprintf("`%s`.`%s`", f.Table, f.Name) }
	addNaming(storepb.SQLReviewRule_NAMING_TABLE, rules.NamingTable, code.NamingTableConventionMismatch, "naming_payload is required for naming table rule", "table", true, BacktickName, e.PosFinding)
	addNaming(storepb.SQLReviewRule_NAMING_COLUMN, rules.NamingColumn, code.NamingColumnConventionMismatch, "naming_payload is required for naming column rule", "column", true, btTableCol, e.posAbsOrFinding)
	addNaming(storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, rules.NamingAutoIncrementColumn, code.NamingAutoIncrementColumnConventionMismatch, "", "auto_increment column", true, btTableCol, e.posAbsOrFinding)
	addNaming(storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, rules.TableDropNaming, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", "drop table", false, BacktickName, e.posAbsOrFinding)

	// Template naming rules.
	invalidTemplateContent := func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("%q meet internal error %q", trimmedStmtText(stmt.Text), f.Detail)
	}
	if rules.NamingForeignKey != nil {
		advisors[storepb.SQLReviewRule_NAMING_INDEX_FK] = e.TemplateNaming(V1TemplateNamingConfig[N]{
			FnNoSchema: rules.NamingForeignKey, Code: code.NamingFKConventionMismatch, Kind: "Foreign key",
			InvalidContent: invalidTemplateContent, NilInvalidPos: true, BacktickQuote: true, Pos: e.PosFinding,
		})
	}
	if rules.NamingIndex != nil {
		errTitle := "Internal error for unique key naming convention rule"
		if e.tidbQuirks {
			errTitle = ""
		}
		advisors[storepb.SQLReviewRule_NAMING_INDEX_IDX] = e.TemplateNaming(V1TemplateNamingConfig[N]{
			FnNoErr: rules.NamingIndex, Code: code.NamingIndexConventionMismatch, Kind: "Index", ErrTitle: errTitle,
			InvalidContent: invalidTemplateContent, NilInvalidPos: true, BacktickQuote: true, Pos: e.PosFinding,
		})
	}
	if rules.NamingUniqueKey != nil {
		advisors[storepb.SQLReviewRule_NAMING_INDEX_UK] = e.TemplateNaming(V1TemplateNamingConfig[N]{
			FnNoErr: rules.NamingUniqueKey, Code: code.NamingUKConventionMismatch, Kind: "Unique key",
			InvalidContent: invalidTemplateContent, NilInvalidPos: true, BacktickQuote: true, Pos: e.PosFinding,
		})
	}

	// Custom per-statement rules.
	addDMLClause := func(ruleType storepb.SQLReviewRule_Type, fn func([]N) []v1review.Finding, codes map[string]code.Code, format string) {
		if fn != nil {
			advisors[ruleType] = e.StmtNodes(fn, func(stmt V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				return codes[f.Name], fmt.Sprintf(format, e.stmtText(stmt, false)), true
			}, e.posFindingOrTiDBFirstToken)
		}
	}
	addDMLClause(storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, rules.StatementDisallowLimit, dmlLimitCodes, "LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses")
	addDMLClause(storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, rules.StatementDisallowOrderBy, dmlOrderByCodes, "ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses")
	if rules.StatementDisallowTruncate != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE] = e.StmtNodes(rules.StatementDisallowTruncate,
			func(_ V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				switch f.Code {
				case v1review.Truncate:
					return code.StatementDisallowTruncate, fmt.Sprintf(`TRUNCATE TABLE %q is not allowed: TRUNCATE auto-commits (breaking surrounding transactional work), bypasses triggers, and resets AUTO_INCREMENT. Prior-backup treats this as DDL and does not produce row-level snapshots.`, f.Table), true
				case v1review.TruncatePartition:
					return code.StatementDisallowTruncate, fmt.Sprintf(`ALTER TABLE %q TRUNCATE PARTITION %q is not allowed: partition truncate shares the auto-commit and prior-backup gaps of TRUNCATE TABLE on MySQL.`, f.Table, f.Name), true
				default:
					return 0, "", false
				}
			}, e.PosFinding)
	}
	if rules.TableNoForeignKey != nil {
		advisors[storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY] = e.StmtNodes(rules.TableNoForeignKey,
			func(_ V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != "" && f.Code != v1review.TableForeignKey {
					return 0, "", false
				}
				return code.TableHasFK, fmt.Sprintf("Foreign key is not allowed in the table `%s`", cmp.Or(f.Table, f.Name)), true
			}, e.posBaseLineTwice)
	}

	// Batch rules.
	if rules.ColumnCurrentTimeCountLimit != nil {
		advisors[storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT] = e.BatchNodes(rules.ColumnCurrentTimeCountLimit,
			func(stmts []V1Stmt[N], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				switch f.Code {
				case v1review.ColumnDefaultCurrentTimeCount:
					return code.DefaultCurrentTimeColumnCountExceedsLimit, fmt.Sprintf("Table `%s` has %s DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, 2), e.posAbsOrFinding(stmts[f.Statement], f), true
				case v1review.ColumnOnUpdateCurrentTimeCount:
					return code.OnUpdateCurrentTimeColumnCountExceedsLimit, fmt.Sprintf("Table `%s` has %s ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, 1), e.posAbsOrFinding(stmts[f.Statement], f), true
				default:
					return 0, "", nil, false
				}
			})
	}
	if fn := rules.SchemaBackwardCompatibility; fn != nil {
		advisors[storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY] = e.BackwardCompatibility(
			func(_ Context, nodes []N) []v1review.Finding { return fn(nodes) },
			nil,
			func(stmt V1Stmt[N], _ v1review.Finding, _ string) string {
				return fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", e.stmtText(stmt, true))
			},
			e.posBaseLineTwice,
		)
	}
	addSimpleBatch(storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, rules.StatementMaxExecutionTime, code.StatementNoMaxExecutionTime, FmtFindingName("The %s is not set"), e.PosNil)
	addSimpleBatch(storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, rules.StatementMergeAlterTable, code.StatementRedundantAlterTable, func(f v1review.Finding) string {
		return fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table)
	}, e.posFindingOrTiDBFirstToken)
	addSimpleBatch(storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX, rules.TableNoDuplicateIndex, code.DuplicateIndexInTable, func(f v1review.Finding) string {
		return fmt.Sprintf("`%s` has duplicate index `%s`", f.Table, f.Name)
	}, e.PosFinding)

	// Shared WHERE / schema-batch rules.
	if fn := rules.StatementWhereDisallowFunctionsAndCalc; fn != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS] = e.WhereDisallowFuncAndCalc(
			func(checkCtx Context) func([]N) []v1review.Finding {
				schema := e.SchemaOf(checkCtx)
				return func(nodes []N) []v1review.Finding { return fn(nodes, schema) }
			},
			false,
			e.PosFinding,
		)
	}
	addOriginalSchemaBatch(storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, rules.IndexPKTypeLimit, code.IndexPKType, func(f v1review.Finding) string {
		return fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", f.Table, f.Name, f.Value)
	})
	addOriginalSchemaBatch(storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, rules.IndexTypeNoBlob, code.IndexTypeNoBlob, func(f v1review.Finding) string {
		return fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", f.Table, f.Name, f.Value)
	})
	if fn := rules.IndexPrimaryKeyTypeAllowlist; fn != nil {
		advisors[storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST] = e.SchemaBatch(code.IndexPKType, func(_ Context, f v1review.Finding) string {
			return fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", f.Name, f.Table, f.Value)
		}, func(checkCtx Context, nodes []N) ([]v1review.Finding, *model.DatabaseMetadata, error) {
			list, err := requireStringArrayPayload(checkCtx.Rule, "string_array_payload is required for this rule")
			if err != nil {
				return nil, nil, err
			}
			findings, err := fn(nodes, list, V1SchemaOf(checkCtx.OriginalMetadata))
			return findings, checkCtx.OriginalMetadata, err
		}, e.PosFinding)
	}
	if fn := rules.IndexTotalNumberLimit; fn != nil {
		advisors[storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT] = e.IndexTotalNumberLimit(fn, func(f v1review.Finding, limit int) string {
			return fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %s", f.Table, limit, f.Value)
		}, e.PosFinding)
	}

	// Specialized and live rules.
	if rules.MaxStatementsInTransaction {
		advisors[storepb.SQLReviewRule_STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION] = mysqlMaxStatementsInTransaction()
	}
	if rules.OnlineMigration != nil {
		advisors[storepb.SQLReviewRule_ADVICE_ONLINE_MIGRATION] = mysqlOnlineMigration(e, engine, rules.OnlineMigration)
	}
	if rules.InsertRowLimit != nil && rules.InsertRowLimitProbes != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT] = e.RowLimit(V1RowLimitConfig[N]{
			Probes:       rules.InsertRowLimitProbes,
			Judge:        rules.InsertRowLimit,
			ExceededCode: code.InsertTooManyRows,
			ExceededVerb: "inserts",
			TrimEstimate: true,
		})
	}
	if rules.AffectedRowLimit != nil && rules.AffectedRowLimitProbes != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT] = e.RowLimit(V1RowLimitConfig[N]{
			Probes:       rules.AffectedRowLimitProbes,
			Judge:        rules.AffectedRowLimit,
			ExceededCode: code.StatementAffectedRowExceedsLimit,
			ExceededVerb: "affected",
		})
	}
	if rules.DMLDryRun != nil && rules.DMLDryRunProbes != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN] = e.DMLDryRun(V1DMLDryRunConfig[N]{
			Probes: rules.DMLDryRunProbes,
			Judge:  rules.DMLDryRun,
		})
	}
	if rules.ColumnComment != nil {
		advisors[storepb.SQLReviewRule_COLUMN_COMMENT] = mysqlColumnComment(e, rules.ColumnComment)
	}
	if fn := rules.ColumnDisallowChangeType; fn != nil {
		advisors[storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE] = e.StmtCtx(
			func(_ context.Context, checkCtx Context, nodes []N) (func([]N) []v1review.Finding, error) {
				guardNilMetadata(checkCtx.OriginalMetadata, nodes, opt.ReadsColumnType)
				schema := V1SchemaOf(checkCtx.OriginalMetadata)
				return func(stmtNodes []N) []v1review.Finding { return fn(stmtNodes, schema) }, nil
			},
			func(_ Context, stmt V1Stmt[N], _ v1review.Finding) (code.Code, string, bool) {
				return code.ChangeColumnType, fmt.Sprintf("\"%s\" changes column type", e.stmtText(stmt, false)), true
			},
			e.posAbsOrFinding,
		)
	}
	if fn := rules.ColumnDisallowDropInIndex; fn != nil {
		advisors[storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX] = e.SimpleBatch(
			func(checkCtx Context, nodes []N) []v1review.Finding {
				guardNilMetadata(checkCtx.OriginalMetadata, nodes, opt.DropsColumn)
				return fn(nodes, V1SchemaOf(checkCtx.OriginalMetadata))
			},
			code.DropIndexColumn, "",
			func(f v1review.Finding) string {
				return fmt.Sprintf("`%s`.`%s` cannot drop index column", f.Table, f.Name)
			},
			e.posFindingOrTiDBFirstToken,
		)
	}
	if rules.ColumnNoNull != nil {
		advisors[storepb.SQLReviewRule_COLUMN_NO_NULL] = mysqlColumnNoNull(e, opt.DefinesColumn, rules.ColumnNoNull)
	}
	if fn := rules.ColumnRequired; fn != nil {
		advisors[storepb.SQLReviewRule_COLUMN_REQUIRED] = e.ColumnRequired(V1ColumnRequiredConfig[N]{
			Fn:        fn,
			Batch:     true,
			SameGroup: func(a, b v1review.Finding) bool { return a.Table == b.Table },
			TableFmt:  "Table `%s` requires columns: %s",
			Pos: func(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
				if e.tidbQuirks {
					return e.PosStatement(stmt, f)
				}
				return &storepb.Position{Line: int32(stmt.BaseLine) + e.findingLine(stmt.Text, f)}
			},
		})
	}
	if fn := rules.DatabaseDropEmptyDatabase; fn != nil {
		advisors[storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE] = e.StmtCtx(
			func(_ context.Context, checkCtx Context, _ []N) (func([]N) []v1review.Finding, error) {
				schema := V1SchemaOf(checkCtx.OriginalMetadata)
				return func(stmtNodes []N) []v1review.Finding { return fn(stmtNodes, schema) }, nil
			},
			func(checkCtx Context, _ V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				switch f.Code {
				case v1review.DatabaseNotCurrent:
					return code.NotCurrentDatabase, fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", f.Name, checkCtx.OriginalMetadata.DatabaseName()), true
				case v1review.DatabaseNotEmpty:
					return code.DatabaseNotEmpty, fmt.Sprintf("Database `%s` is not allowed to drop if not empty", f.Name), true
				default:
					return 0, "", false
				}
			},
			e.posBaseLineTwice,
		)
	}
	if fn := rules.TableComment; fn != nil {
		advisors[storepb.SQLReviewRule_TABLE_COMMENT] = e.StmtCtx(
			func(_ context.Context, checkCtx Context, _ []N) (func([]N) []v1review.Finding, error) {
				var tableCfg *v1review.TableCommentConfig
				if payload := checkCtx.Rule.GetCommentConventionPayload(); payload != nil {
					tableCfg = &v1review.TableCommentConfig{Required: payload.Required, MaxLength: int(payload.MaxLength)}
				}
				return func(stmtNodes []N) []v1review.Finding { return fn(stmtNodes, tableCfg) }, nil
			},
			func(checkCtx Context, _ V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				switch f.Code {
				case v1review.TableCommentMissing:
					return code.CommentEmpty, fmt.Sprintf("Table `%s` requires comments", f.Name), true
				case v1review.TableCommentTooLong:
					return code.CommentTooLong, fmt.Sprintf("The length of table `%s` comment should be within %d characters", f.Name, checkCtx.Rule.GetCommentConventionPayload().GetMaxLength()), true
				default:
					return 0, "", false
				}
			},
			e.PosStatement,
		)
	}
	if fn := rules.TableLimitSize; fn != nil {
		advisors[storepb.SQLReviewRule_TABLE_LIMIT_SIZE] = e.SchemaBatch(code.TableExceedLimitSize,
			func(_ Context, f v1review.Finding) string {
				return fmt.Sprintf("Apply DDL on large table '%s' ( %s rows ) will lock table for a long time", f.Name, f.Value)
			},
			func(checkCtx Context, nodes []N) ([]v1review.Finding, *model.DatabaseMetadata, error) {
				limit, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
				if err != nil {
					return nil, nil, err
				}
				return fn(nodes, checkCtx.DBSchema, limit), nil, nil
			},
			e.PosStatement,
		)
	}
	if fn := rules.TableRequirePK; fn != nil {
		advisors[storepb.SQLReviewRule_TABLE_REQUIRE_PK] = e.Batch(
			func(checkCtx Context, nodes []N) []v1review.Finding {
				return fn(nodes, V1SchemaPtrOf(checkCtx.OriginalMetadata))
			},
			func(stmts []V1Stmt[N], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				if f.Code != v1review.TablePrimaryKeyMissing {
					return 0, "", nil, false
				}
				pos := base.LinePosition(0)
				if f.Statement >= 0 {
					pos = e.posAbsOrFinding(stmts[f.Statement], f)
				}
				return code.TableNoPK, fmt.Sprintf("Table `%s` requires PRIMARY KEY", f.Name), pos, true
			},
		)
	}
	if fn := rules.TableTextFieldsTotalLength; fn != nil {
		advisors[storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH] = e.StmtCtx(
			func(_ context.Context, checkCtx Context, _ []N) (func([]N) []v1review.Finding, error) {
				maximum, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
				if err != nil {
					return nil, err
				}
				final := V1SchemaPtrOf(checkCtx.FinalMetadata)
				return func(stmtNodes []N) []v1review.Finding { return fn(stmtNodes, final, maximum) }, nil
			},
			func(checkCtx Context, stmt V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != v1review.TableTextTooLong {
					return 0, "", false
				}
				adviceCode := code.TotalTextLengthExceedsLimit
				if opt.IsCreateTable != nil && opt.IsCreateTable(stmt.Node) {
					adviceCode = code.IndexCountExceedsLimit
				}
				return adviceCode, fmt.Sprintf("Table %q total text column length (%s) exceeds the limit (%d).", f.Name, f.Value, checkCtx.Rule.GetNumberPayload().GetNumber()), true
			},
			func(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
				return &storepb.Position{Line: int32(stmt.BaseLine) + e.findingLine(stmt.Text, f)}
			},
		)
	}

	return advisors
}

func guardNilMetadata[N any](md *model.DatabaseMetadata, nodes []N, pred func(N) bool) {
	// Preserves V1 panic parity when a statement inspects schema metadata with nil metadata.
	if md == nil && pred != nil && slices.ContainsFunc(nodes, pred) {
		md.GetSchemaMetadata("")
	}
}

func mysqlMaxStatementsInTransaction() Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		if _, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level); err != nil {
			return nil, err
		}
		if _, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule"); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func mysqlOnlineMigration[N any](e *V1Engine[N], engine storepb.Engine, fn func([]N, v1review.Schema, string, int64) []v1review.Finding) Func {
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		n, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		minRows := int64(n)
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		if checkCtx.EnableGhost {
			ghostDB := common.BackupDatabaseNameOfEngine(engine)
			if !DatabaseExists(ctx, checkCtx, ghostDB) {
				return []*storepb.Advice{newAdvice(level, code.DatabaseNotExists, title, fmt.Sprintf("Needs database %q to save temporary data for online migration but it does not exist", ghostDB), nil)}, nil
			}
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(nodes, e.SchemaOf(checkCtx), checkCtx.CurrentDatabase, minRows) {
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AdviseOnlineMigrationForStatement.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Estimated table row count of %q is %s exceeding the set value %d. Consider using online migration for this statement", f.Table, f.Value, minRows),
				StartPosition: e.PosFinding(stmt, f),
				EndPosition:   e.PosContentEnd(stmt, f),
			})
		}
		if len(adviceList) > 1 || (len(adviceList) == 1 && len(checkCtx.ParsedStatements) > 1) {
			return adviceList, nil
		}
		if len(adviceList) == 1 && len(checkCtx.ParsedStatements) == 1 {
			if checkCtx.EnableGhost {
				return nil, nil
			}
			adviceList[0].Code = code.AdviseOnlineMigration.Int32()
			return adviceList, nil
		}
		if len(adviceList) == 0 && checkCtx.EnableGhost {
			return []*storepb.Advice{newAdvice(level, code.AdviseNoOnlineMigration, title, "Advise to disable online migration because found no statements that need online migration", nil)}, nil
		}
		return nil, nil
	}
}

func mysqlColumnComment[N any](e *V1Engine[N], fn func([]N, bool, int) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		payload := checkCtx.Rule.GetCommentConventionPayload()
		required, maxLength := true, 0
		if e.tidbQuirks {
			required, maxLength = false, -1
		}
		if payload != nil {
			required, maxLength = payload.Required, int(payload.MaxLength)
		}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i:i+1:i+1], required, maxLength) {
				if payload == nil {
					return nil, errors.New("comment_convention_payload is required for column comment rule")
				}
				var adviceCode code.Code
				var content string
				switch f.Code {
				case v1review.ColumnCommentTooLong:
					if e.tidbQuirks && payload.MaxLength < 0 {
						continue
					}
					adviceCode = code.CommentTooLong
					content = fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", f.Table, f.Name, payload.MaxLength)
				case v1review.ColumnCommentMissing:
					adviceCode = code.CommentEmpty
					content = fmt.Sprintf("Column `%s`.`%s` requires comments", f.Table, f.Name)
				default:
					continue
				}
				adviceList = append(adviceList, newAdvice(status, adviceCode, title, content, e.posAlterFirstToken(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

func mysqlColumnNoNull[N any](e *V1Engine[N], definesColumn func(N) bool, fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		guardNilMetadata(checkCtx.FinalMetadata, nodes, definesColumn)
		type placed struct {
			pos    *storepb.Position
			column string
			table  string
		}
		var columns []placed
		for _, f := range fn(nodes, V1SchemaOf(checkCtx.FinalMetadata)) {
			columns = append(columns, placed{pos: e.posAbsOrFinding(stmts[f.Statement], f), column: f.Name, table: f.Table})
		}
		slices.SortStableFunc(columns, func(a, b placed) int {
			return cmp.Or(cmp.Compare(a.pos.Line, b.pos.Line), cmp.Compare(a.column, b.column))
		})
		var adviceList []*storepb.Advice
		for _, c := range columns {
			adviceList = append(adviceList, newAdvice(status, code.ColumnCannotNull, title, fmt.Sprintf("`%s`.`%s` cannot have NULL value", c.table, c.column), c.pos))
		}
		return adviceList, nil
	}
}
