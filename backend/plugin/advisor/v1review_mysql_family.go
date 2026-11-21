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
	IsRenameIndex       func(N) bool
	IsDropDatabase      func(N) bool
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
	e := NewV1EngineWithOptions(V1EngineOptions[N]{
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
	mysqlCompatEntries = map[v1review.Code]V1CompatEntry{
		v1review.IncompatibleDropDatabase:  {Code: code.CompatibilityDropDatabase},
		v1review.IncompatibleRenameTable:   {Code: code.CompatibilityRenameTable},
		v1review.IncompatibleDropTable:     {Code: code.CompatibilityDropTable},
		v1review.IncompatibleRenameColumn:  {Code: code.CompatibilityRenameColumn},
		v1review.IncompatibleDropColumn:    {Code: code.CompatibilityDropColumn},
		v1review.IncompatibleAddPrimaryKey: {Code: code.CompatibilityAddPrimaryKey},
		v1review.IncompatibleAddUniqueKey:  {Code: code.CompatibilityAddUniqueKey},
		v1review.IncompatibleAddForeignKey: {Code: code.CompatibilityAddForeignKey},
		v1review.IncompatibleAddCheck:      {Code: code.CompatibilityAddCheck},
		v1review.IncompatibleAlterCheck:    {Code: code.CompatibilityAlterCheck},
		v1review.IncompatibleAlterColumn:   {Code: code.CompatibilityAlterColumn},
	}
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
	addDisallowCreate := func(ruleType storepb.SQLReviewRule_Type, fn func([]N) []v1review.Finding, c code.Code, object string) {
		addSimple(ruleType, fn, c, e.posBaseLineTwice, func(stmt V1Stmt[N], _ v1review.Finding) string {
			return fmt.Sprintf("%s is forbidden, but \"%s\" creates", object, strings.TrimSpace(stmt.Text))
		})
	}
	addNumber := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, int) []v1review.Finding, missingMsg string, positiveOnly bool, c code.Code, pos V1PosFunc[N], content func(V1Stmt[N], v1review.Finding, int) string) {
		if fn != nil {
			advisors[ruleType] = e.NumberStmtWithMsg(fn, missingMsg, positiveOnly, c, content, pos)
		}
	}
	addStringArray := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, []string) []v1review.Finding, c code.Code, pos V1PosFunc[N], content func(V1Stmt[N], v1review.Finding) string) {
		if fn != nil {
			advisors[ruleType] = e.StringArrayStmt(fn, "", c, "", content, pos)
		}
	}
	addNaming := func(ruleType storepb.SQLReviewRule_Type, fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error), c code.Code, missingMsg string, pos V1PosFunc[N], mismatch func(v1review.Finding, string) string, tooLong func(v1review.Finding, int) string) {
		if fn != nil {
			advisors[ruleType] = e.NamingRegex(fn, c, missingMsg, mismatch, tooLong, pos)
		}
	}

	// Simple per-statement rules.
	addSimple(storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, rules.ColumnAutoIncrementMustInteger, code.AutoIncrementColumnNotInteger, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", f.Table, f.Name)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, rules.ColumnAutoIncrementMustUnsigned, code.AutoIncrementColumnSigned, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", f.Table, f.Name)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, rules.ColumnDisallowChange, code.UseChangeColumnStatement, e.posStatementAbsOrStmt, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" contains CHANGE COLUMN statement", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, rules.ColumnDisallowChangingOrder, code.ChangeColumnOrder, e.posStatementAbsOrStmt, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" changes column order", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_DROP, rules.ColumnDisallowDrop, code.DropColumn, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("drops column \"%s\" of table \"%s\"", f.Name, f.Table)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, rules.ColumnDisallowSetCharset, code.SetColumnCharset, e.posStatementFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("Disallow set column charset but \"%s\" does", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET, rules.ColumnRequireCharset, code.NoCharset, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column %s does not have a character set specified", f.Name)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION, rules.ColumnRequireCollation, code.NoCollation, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column %s does not have a collation specified", f.Name)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, rules.ColumnRequireDefault, code.NoDefault, e.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column `%s`.`%s` doesn't have DEFAULT.", f.Table, f.Name)
	})
	addSimple(storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, rules.ColumnSetDefaultForNotNull, code.NotNullColumnWithNoDefault, e.posAlterFirstToken, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", f.Table, f.Name)
	})
	addSimple(storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, rules.EngineMySQLUseInnoDB, code.NotInnoDBEngine, e.PosSearchOrStart, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s;\" doesn't use InnoDB engine", trimmedStmtText(stmt.Text))
	})
	addSimple(storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, rules.IndexNoDuplicateColumn, code.DuplicateColumnInIndex, e.PosFinding, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("%s `%s` has duplicate column `%s`.`%s`", f.Value, keyName(stmt.Text, f, e.tidbQuirks), f.Table, f.Other)
	})
	addSimple(storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, rules.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier, e.posSearchOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Identifier %q is a keyword and should be avoided", f.Name)
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION, rules.StatementAddColumnWithoutPosition, code.StatementAddColumnWithPosition, e.PosFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("add column with position \"%s\"", f.Name)
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, rules.StatementDisallowCommit, code.StatementDisallowCommit, e.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, rules.StatementInsertDisallowOrderByRand, code.InsertUseOrderByRand, e.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" uses ORDER BY RAND in the INSERT statement", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, rules.StatementInsertMustSpecifyColumn, code.InsertNotSpecifyColumn, e.posAbsOrFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", e.stmtText(stmt, true))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, rules.StatementRequireAlgorithmOption, code.StatementNoAlgorithmOption, e.PosFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return "ALTER TABLE statement should include " + f.Name + " option"
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, rules.StatementRequireLockOption, code.StatementNoLockOption, e.PosFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return "ALTER TABLE statement should include " + f.Name + " option"
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, rules.StatementSelectNoSelectAll, code.StatementSelectAll, e.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" uses SELECT all", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL, rules.StatementWhereNoEqualNull, code.StatementWhereNoEqualNull, e.PosFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("WHERE clause contains equal null: %s", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, rules.StatementWhereNoLeadingWildcardLike, code.StatementLeadingWildcardLike, e.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" uses leading wildcard LIKE", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, rules.StatementWhereRequireSelect, code.StatementNoWhere, e.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" requires WHERE clause", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, rules.StatementWhereRequireUpdateDelete, code.StatementNoWhere, e.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" requires WHERE clause", e.stmtText(stmt, false))
	})
	addSimple(storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET, rules.TableDisallowSetCharset, code.DisallowSetCharset, e.posStatementFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", strings.TrimSpace(stmt.Text))
	})
	addSimple(storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET, rules.TableRequireCharset, code.NoCharset, e.posStatementFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Table %s does not have a character set specified", f.Name)
	})
	addSimple(storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION, rules.TableRequireCollation, code.NoCollation, e.posStatementFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Table %s does not have a collation specified", f.Name)
	})

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
	addStringArray(storepb.SQLReviewRule_TABLE_DISALLOW_DDL, rules.TableDisallowDDL, code.TableDisallowDDL, e.PosFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("DDL is disallowed on table %s.", f.Name)
	})
	addStringArray(storepb.SQLReviewRule_TABLE_DISALLOW_DML, rules.TableDisallowDML, code.TableDisallowDML, e.PosFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("DML is disallowed on table %s.", f.Name)
	})

	// Regex naming rules.
	addNaming(storepb.SQLReviewRule_NAMING_TABLE, rules.NamingTable, code.NamingTableConventionMismatch, "naming_payload is required for naming table rule", e.PosFinding,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", f.Name, format)
		},
		func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("`%s` mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
		},
	)
	addNaming(storepb.SQLReviewRule_NAMING_COLUMN, rules.NamingColumn, code.NamingColumnConventionMismatch, "naming_payload is required for naming column rule", e.posAbsOrFinding,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", f.Table, f.Name, format)
		},
		func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
		},
	)
	addNaming(storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, rules.NamingAutoIncrementColumn, code.NamingAutoIncrementColumnConventionMismatch, "naming_payload is required for this rule", e.posAbsOrFinding,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, naming format should be %q", f.Table, f.Name, format)
		},
		func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
		},
	)
	addNaming(storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, rules.TableDropNaming, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", e.posAbsOrFinding,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", f.Name, format)
		},
		nil,
	)

	// Template naming rules.
	templateInvalidContent := func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("%q meet internal error %q", trimmedStmtText(stmt.Text), f.Detail)
	}
	renameIndexSchema := func(node N, metadata *model.DatabaseMetadata) v1review.Schema {
		if opt.IsRenameIndex != nil && opt.IsRenameIndex(node) {
			return V1SchemaOf(metadata)
		}
		return v1review.Schema{}
	}
	if rules.NamingForeignKey != nil {
		fn := rules.NamingForeignKey
		advisors[storepb.SQLReviewRule_NAMING_INDEX_FK] = e.TemplateNamingWithOptions(
			func(nodes []N, cfg v1review.TemplateConfig, _ v1review.Schema) ([]v1review.Finding, error) {
				return fn(nodes, cfg)
			},
			code.NamingFKConventionMismatch,
			"Internal error for foreign key naming convention rule",
			"Foreign key",
			"",
			true,
			V1TemplateNamingOptions[N]{BacktickQuote: true, InvalidContent: templateInvalidContent},
			e.PosFinding,
		)
	}
	if rules.NamingIndex != nil {
		fn := rules.NamingIndex
		errTitle := "Internal error for unique key naming convention rule"
		if e.tidbQuirks {
			errTitle = "Internal error for index naming convention rule"
		}
		advisors[storepb.SQLReviewRule_NAMING_INDEX_IDX] = e.TemplateNamingWithOptions(
			func(nodes []N, cfg v1review.TemplateConfig, schema v1review.Schema) ([]v1review.Finding, error) {
				return fn(nodes, cfg, schema), nil
			},
			code.NamingIndexConventionMismatch,
			errTitle,
			"Index",
			"",
			true,
			V1TemplateNamingOptions[N]{BacktickQuote: true, InvalidContent: templateInvalidContent, SchemaFor: renameIndexSchema},
			e.PosFinding,
		)
	}
	if rules.NamingUniqueKey != nil {
		fn := rules.NamingUniqueKey
		advisors[storepb.SQLReviewRule_NAMING_INDEX_UK] = e.TemplateNamingWithOptions(
			func(nodes []N, cfg v1review.TemplateConfig, schema v1review.Schema) ([]v1review.Finding, error) {
				return fn(nodes, cfg, schema), nil
			},
			code.NamingUKConventionMismatch,
			"Internal error for unique key naming convention rule",
			"Unique key",
			"",
			true,
			V1TemplateNamingOptions[N]{BacktickQuote: true, InvalidContent: templateInvalidContent, SchemaFor: renameIndexSchema},
			e.PosFinding,
		)
	}

	// Custom per-statement rules using V1Engine.StmtNodes.
	if rules.StatementDisallowLimit != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT] = e.StmtNodes(rules.StatementDisallowLimit,
			func(stmt V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				return dmlLimitCodes[f.Name], fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", e.stmtText(stmt, false)), true
			}, e.posFindingOrTiDBFirstToken)
	}
	if rules.StatementDisallowOrderBy != nil {
		advisors[storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY] = e.StmtNodes(rules.StatementDisallowOrderBy,
			func(stmt V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				return dmlOrderByCodes[f.Name], fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", e.stmtText(stmt, false)), true
			}, e.posFindingOrTiDBFirstToken)
	}
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
	if rules.TableDisallowPartition != nil {
		advisors[storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION] = e.StmtNodes(rules.TableDisallowPartition,
			func(stmt V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != "" && f.Code != v1review.TablePartitioned {
					return 0, "", false
				}
				return code.CreateTablePartition, fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", e.stmtText(stmt, false)), true
			}, e.posStatementFinding)
	}
	if rules.TableNoForeignKey != nil {
		advisors[storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY] = e.StmtNodes(rules.TableNoForeignKey,
			func(_ V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != "" && f.Code != v1review.TableForeignKey {
					return 0, "", false
				}
				table := f.Table
				if table == "" {
					table = f.Name
				}
				return code.TableHasFK, fmt.Sprintf("Foreign key is not allowed in the table `%s`", table), true
			}, e.posBaseLineTwice)
	}

	// Batch rules using V1Engine.Batch / BackwardCompatibility.
	if rules.ColumnCurrentTimeCountLimit != nil {
		fn := rules.ColumnCurrentTimeCountLimit
		advisors[storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT] = e.Batch(func(_ Context, nodes []N) []v1review.Finding { return fn(nodes) },
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
	if rules.SchemaBackwardCompatibility != nil {
		fn := rules.SchemaBackwardCompatibility
		advisors[storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY] = e.BackwardCompatibility(
			func(_ Context, nodes []N) []v1review.Finding { return fn(nodes) },
			mysqlCompatEntries,
			func(stmt V1Stmt[N], _ v1review.Finding, _ string) string {
				return fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", e.stmtText(stmt, true))
			},
			e.posBaseLineTwice,
		)
	}
	if rules.StatementMaxExecutionTime != nil {
		fn := rules.StatementMaxExecutionTime
		advisors[storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME] = e.Batch(func(_ Context, nodes []N) []v1review.Finding { return fn(nodes) },
			func(_ []V1Stmt[N], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				return code.StatementNoMaxExecutionTime, fmt.Sprintf("The %s is not set", f.Name), nil, true
			})
	}
	if rules.StatementMergeAlterTable != nil {
		fn := rules.StatementMergeAlterTable
		advisors[storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE] = e.Batch(func(_ Context, nodes []N) []v1review.Finding { return fn(nodes) },
			func(stmts []V1Stmt[N], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				return code.StatementRedundantAlterTable, fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table), e.posFindingOrTiDBFirstToken(stmts[f.Statement], f), true
			})
	}
	if rules.TableNoDuplicateIndex != nil {
		fn := rules.TableNoDuplicateIndex
		advisors[storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX] = e.Batch(func(_ Context, nodes []N) []v1review.Finding { return fn(nodes) },
			func(stmts []V1Stmt[N], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				return code.DuplicateIndexInTable, fmt.Sprintf("`%s` has duplicate index `%s`", f.Table, f.Name), e.PosFinding(stmts[f.Statement], f), true
			})
	}

	// Shared WHERE / index-schema rules.
	if rules.StatementWhereDisallowFunctionsAndCalc != nil {
		fn := rules.StatementWhereDisallowFunctionsAndCalc
		advisors[storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS] = e.WhereDisallowFuncAndCalc(
			func(checkCtx Context) func([]N) []v1review.Finding {
				schema := e.SchemaOf(checkCtx)
				return func(nodes []N) []v1review.Finding { return fn(nodes, schema) }
			},
			false,
			e.PosFinding,
		)
	}
	if rules.IndexPKTypeLimit != nil {
		advisors[storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT] = mysqlIndexSchemaBatch(e, rules.IndexPKTypeLimit, code.IndexPKType, func(f v1review.Finding) string {
			return fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", f.Table, f.Name, f.Value)
		})
	}
	if rules.IndexTypeNoBlob != nil {
		advisors[storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB] = mysqlIndexSchemaBatch(e, rules.IndexTypeNoBlob, code.IndexTypeNoBlob, func(f v1review.Finding) string {
			return fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", f.Table, f.Name, f.Value)
		})
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
	if rules.ColumnDisallowChangeType != nil {
		advisors[storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE] = mysqlColumnDisallowChangeType(e, opt.ReadsColumnType, rules.ColumnDisallowChangeType)
	}
	if rules.ColumnDisallowDropInIndex != nil {
		advisors[storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX] = mysqlColumnDisallowDropInIndex(e, opt.DropsColumn, rules.ColumnDisallowDropInIndex)
	}
	if rules.ColumnNoNull != nil {
		advisors[storepb.SQLReviewRule_COLUMN_NO_NULL] = mysqlColumnNoNull(e, opt.DefinesColumn, rules.ColumnNoNull)
	}
	if rules.ColumnRequired != nil {
		advisors[storepb.SQLReviewRule_COLUMN_REQUIRED] = mysqlColumnRequired(e, rules.ColumnRequired)
	}
	if rules.DatabaseDropEmptyDatabase != nil {
		advisors[storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE] = mysqlDatabaseDropEmptyDatabase(e, opt.IsDropDatabase, rules.DatabaseDropEmptyDatabase)
	}
	if rules.IndexPrimaryKeyTypeAllowlist != nil {
		advisors[storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST] = mysqlIndexPrimaryKeyTypeAllowlist(e, rules.IndexPrimaryKeyTypeAllowlist)
	}
	if rules.IndexTotalNumberLimit != nil {
		advisors[storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT] = mysqlIndexTotalNumberLimit(e, rules.IndexTotalNumberLimit)
	}
	if rules.TableComment != nil {
		advisors[storepb.SQLReviewRule_TABLE_COMMENT] = mysqlTableComment(e, rules.TableComment)
	}
	if rules.TableLimitSize != nil {
		advisors[storepb.SQLReviewRule_TABLE_LIMIT_SIZE] = mysqlTableLimitSize(e, rules.TableLimitSize)
	}
	if rules.TableRequirePK != nil {
		advisors[storepb.SQLReviewRule_TABLE_REQUIRE_PK] = mysqlTableRequirePK(e, rules.TableRequirePK)
	}
	if rules.TableTextFieldsTotalLength != nil {
		advisors[storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH] = mysqlTableTextFieldsTotalLength(e, opt.IsCreateTable, rules.TableTextFieldsTotalLength)
	}

	return advisors
}

func guardNilMetadata[N any](md *model.DatabaseMetadata, nodes []N, pred func(N) bool) {
	// Preserves V1 panic parity when a statement inspects schema metadata with nil metadata.
	if md == nil && pred != nil && slices.ContainsFunc(nodes, pred) {
		md.GetSchemaMetadata("")
	}
}

func mysqlSchemaBatch[N any](
	e *V1Engine[N],
	adviceCode code.Code,
	contentFn func(v1review.Finding) string,
	eval func(Context, []N) ([]v1review.Finding, *model.DatabaseMetadata, error),
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		findings, missingMD, err := eval(checkCtx, nodes)
		if err != nil {
			ReadMissingSchema(missingMD, err)
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, newAdvice(status, adviceCode, title, contentFn(f), e.PosFinding(stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}

func mysqlIndexSchemaBatch[N any](
	e *V1Engine[N],
	fn func([]N, v1review.Schema) ([]v1review.Finding, error),
	adviceCode code.Code,
	contentFn func(v1review.Finding) string,
) Func {
	return mysqlSchemaBatch(e, adviceCode, contentFn, func(checkCtx Context, nodes []N) ([]v1review.Finding, *model.DatabaseMetadata, error) {
		findings, err := fn(nodes, V1SchemaOf(checkCtx.OriginalMetadata))
		return findings, checkCtx.OriginalMetadata, err
	})
}

func mysqlMaxStatementsInTransaction() Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		if _, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level); err != nil {
			return nil, err
		}
		if _, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule"); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func mysqlOnlineMigration[N any](e *V1Engine[N], engine storepb.Engine, fn func([]N, v1review.Schema, string, int64) []v1review.Finding) Func {
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		n, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
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
		schema := e.SchemaOf(checkCtx)
		var adviceList []*storepb.Advice
		for _, f := range fn(nodes, schema, checkCtx.CurrentDatabase, minRows) {
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AdviseOnlineMigrationForStatement.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Estimated table row count of %q is %s exceeding the set value %d. Consider using online migration for this statement", f.Table, f.Value, minRows),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.findingLine(stmt.Text, f))),
				EndPosition:   base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.ContentEndLine(stmt.Text))),
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

func mysqlColumnDisallowChangeType[N any](e *V1Engine[N], readsColumnType func(N) bool, fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		guardNilMetadata(checkCtx.OriginalMetadata, nodes, readsColumnType)
		schema := V1SchemaOf(checkCtx.OriginalMetadata)
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i:i+1:i+1], schema) {
				adviceList = append(adviceList, newAdvice(status, code.ChangeColumnType, title, fmt.Sprintf("\"%s\" changes column type", e.stmtText(stmt, false)), e.posAbsOrFinding(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

func mysqlColumnDisallowDropInIndex[N any](e *V1Engine[N], dropsColumn func(N) bool, fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		guardNilMetadata(checkCtx.OriginalMetadata, nodes, dropsColumn)
		var adviceList []*storepb.Advice
		for _, f := range fn(nodes, V1SchemaOf(checkCtx.OriginalMetadata)) {
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, newAdvice(status, code.DropIndexColumn, title, fmt.Sprintf("`%s`.`%s` cannot drop index column", f.Table, f.Name), e.posFindingOrTiDBFirstToken(stmt, f)))
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
			line   int
			column string
			table  string
		}
		var columns []placed
		for _, f := range fn(nodes, V1SchemaOf(checkCtx.FinalMetadata)) {
			stmt := stmts[f.Statement]
			line := stmt.BaseLine + int(e.findingLine(stmt.Text, f))
			if e.tidbQuirks {
				line = e.absLine(stmt, f.Start)
			}
			columns = append(columns, placed{line: line, column: f.Name, table: f.Table})
		}
		slices.SortStableFunc(columns, func(a, b placed) int {
			return cmp.Or(cmp.Compare(a.line, b.line), cmp.Compare(a.column, b.column))
		})
		var adviceList []*storepb.Advice
		for _, c := range columns {
			adviceList = append(adviceList, newAdvice(status, code.ColumnCannotNull, title, fmt.Sprintf("`%s`.`%s` cannot have NULL value", c.table, c.column), base.ConvertANTLRLineToPosition(c.line)))
		}
		return adviceList, nil
	}
}

func mysqlColumnRequired[N any](e *V1Engine[N], fn func([]N, []string) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		list, err := RequireStringArrayPayload(checkCtx.Rule, "string_array_payload is required for column required rule")
		if err != nil {
			return nil, err
		}
		findings := fn(nodes, list)
		var adviceList []*storepb.Advice
		for i := 0; i < len(findings); {
			f := findings[i]
			var missing []string
			for ; i < len(findings) && findings[i].Table == f.Table; i++ {
				missing = append(missing, findings[i].Name)
			}
			stmt := stmts[f.Statement]
			pos := e.posStatement(stmt)
			if !e.tidbQuirks {
				pos = &storepb.Position{Line: int32(stmt.BaseLine + int(e.findingLine(stmt.Text, f)))}
			}
			adviceList = append(adviceList, newAdvice(status, code.NoRequiredColumn, title, fmt.Sprintf("Table `%s` requires columns: %s", f.Table, strings.Join(missing, ", ")), pos))
		}
		return adviceList, nil
	}
}

func mysqlDatabaseDropEmptyDatabase[N any](e *V1Engine[N], isDropDatabase func(N) bool, fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			var schema v1review.Schema
			if isDropDatabase != nil && isDropDatabase(stmt.Node) {
				schema = V1SchemaOf(checkCtx.OriginalMetadata)
			}
			for _, f := range fn(nodes[i:i+1:i+1], schema) {
				switch f.Code {
				case v1review.DatabaseNotCurrent:
					adviceList = append(adviceList, newAdvice(status, code.NotCurrentDatabase, title, fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", f.Name, checkCtx.OriginalMetadata.DatabaseName()), e.posBaseLineTwice(stmt, f)))
				case v1review.DatabaseNotEmpty:
					adviceList = append(adviceList, newAdvice(status, code.DatabaseNotEmpty, title, fmt.Sprintf("Database `%s` is not allowed to drop if not empty", f.Name), e.posBaseLineTwice(stmt, f)))
				default:
				}
			}
		}
		return adviceList, nil
	}
}

func mysqlIndexPrimaryKeyTypeAllowlist[N any](e *V1Engine[N], fn func([]N, []string, v1review.Schema) ([]v1review.Finding, error)) Func {
	return mysqlSchemaBatch(e, code.IndexPKType, func(f v1review.Finding) string {
		return fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", f.Name, f.Table, f.Value)
	}, func(checkCtx Context, nodes []N) ([]v1review.Finding, *model.DatabaseMetadata, error) {
		list, err := RequireStringArrayPayload(checkCtx.Rule, "string_array_payload is required for this rule")
		if err != nil {
			return nil, nil, err
		}
		findings, err := fn(nodes, list, V1SchemaOf(checkCtx.OriginalMetadata))
		return findings, checkCtx.OriginalMetadata, err
	})
}

func mysqlIndexTotalNumberLimit[N any](e *V1Engine[N], fn func([]N, int, v1review.Schema) ([]v1review.Finding, error)) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		limit, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		return mysqlSchemaBatch(e, code.IndexCountExceedsLimit, func(f v1review.Finding) string {
			return fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %s", f.Table, limit, f.Value)
		}, func(checkCtx Context, nodes []N) ([]v1review.Finding, *model.DatabaseMetadata, error) {
			findings, err := fn(nodes, limit, V1SchemaOf(checkCtx.FinalMetadata))
			return findings, checkCtx.FinalMetadata, err
		})(context.Background(), checkCtx)
	}
}

func mysqlTableComment[N any](e *V1Engine[N], fn func([]N, *v1review.TableCommentConfig) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var tableCfg *v1review.TableCommentConfig
		if payload := checkCtx.Rule.GetCommentConventionPayload(); payload != nil {
			tableCfg = &v1review.TableCommentConfig{Required: payload.Required, MaxLength: int(payload.MaxLength)}
		}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			pos := e.posStatement(stmt)
			for _, f := range fn(nodes[i:i+1:i+1], tableCfg) {
				switch f.Code {
				case v1review.TableCommentMissing:
					adviceList = append(adviceList, newAdvice(status, code.CommentEmpty, title, fmt.Sprintf("Table `%s` requires comments", f.Name), pos))
				case v1review.TableCommentTooLong:
					adviceList = append(adviceList, newAdvice(status, code.CommentTooLong, title, fmt.Sprintf("The length of table `%s` comment should be within %d characters", f.Name, tableCfg.MaxLength), pos))
				default:
				}
			}
		}
		return adviceList, nil
	}
}

func mysqlTableLimitSize[N any](e *V1Engine[N], fn func([]N, *metadatapb.DatabaseSchemaMetadata, int) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		limit, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(nodes, checkCtx.DBSchema, limit) {
			if f.Code == v1review.TableTooLarge {
				adviceList = append(adviceList, newAdvice(status, code.TableExceedLimitSize, title, fmt.Sprintf("Apply DDL on large table '%s' ( %s rows ) will lock table for a long time", f.Name, f.Value), e.posStatement(stmts[f.Statement])))
			}
		}
		return adviceList, nil
	}
}

func mysqlTableRequirePK[N any](e *V1Engine[N], fn func([]N, *v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(nodes, V1SchemaPtrOf(checkCtx.OriginalMetadata)) {
			if f.Code != v1review.TablePrimaryKeyMissing {
				continue
			}
			line := 0
			if f.Statement >= 0 {
				stmt := stmts[f.Statement]
				if e.tidbQuirks {
					line = e.absLine(stmt, f.Start)
				} else {
					line = stmt.BaseLine + int(e.findingLine(stmt.Text, f))
				}
			}
			adviceList = append(adviceList, newAdvice(status, code.TableNoPK, title, fmt.Sprintf("Table `%s` requires PRIMARY KEY", f.Name), base.ConvertANTLRLineToPosition(line)))
		}
		return adviceList, nil
	}
}

func mysqlTableTextFieldsTotalLength[N any](e *V1Engine[N], isCreateTable func(N) bool, fn func([]N, *v1review.Schema, int) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		maximum, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		final := V1SchemaPtrOf(checkCtx.FinalMetadata)
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			adviceCode := code.TotalTextLengthExceedsLimit
			if isCreateTable != nil && isCreateTable(stmt.Node) {
				adviceCode = code.IndexCountExceedsLimit
			}
			for _, f := range fn(nodes[i:i+1:i+1], final, maximum) {
				if f.Code == v1review.TableTextTooLong {
					adviceList = append(adviceList, newAdvice(status, adviceCode, title, fmt.Sprintf("Table %q total text column length (%s) exceeds the limit (%d).", f.Name, f.Value, maximum), &storepb.Position{Line: int32(stmt.BaseLine) + e.findingLine(stmt.Text, f)}))
				}
			}
		}
		return adviceList, nil
	}
}
