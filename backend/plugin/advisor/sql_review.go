package advisor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// How to add a SQL review rule:
//   1. Implement an advisor.(plugin/advisor/mysql or plugin/advisor/pg)
//   2. Register this advisor in map[storepb.Engine][AdvisorType].(plugin/advisor.go)
//   3. Add advisor error code if needed(plugin/advisor/code.go).
//   4. Map SQLReviewRuleType to advisor.Type in getAdvisorTypeByRule(current file).

// SQLReviewRuleType is the type of schema rule.
type SQLReviewRuleType string

const (
	// SchemaRuleMySQLEngine require InnoDB as the storage engine.
	SchemaRuleMySQLEngine SQLReviewRuleType = "engine.mysql.use-innodb"

	// SchemaRuleFullyQualifiedObjectName enforces using fully qualified object name.
	SchemaRuleFullyQualifiedObjectName SQLReviewRuleType = "naming.fully-qualified"
	// SchemaRuleTableNaming enforce the table name format.
	SchemaRuleTableNaming SQLReviewRuleType = "naming.table"
	// SchemaRuleColumnNaming enforce the column name format.
	SchemaRuleColumnNaming SQLReviewRuleType = "naming.column"
	// SchemaRulePKNaming enforce the primary key name format.
	SchemaRulePKNaming SQLReviewRuleType = "naming.index.pk"
	// SchemaRuleUKNaming enforce the unique key name format.
	SchemaRuleUKNaming SQLReviewRuleType = "naming.index.uk"
	// SchemaRuleFKNaming enforce the foreign key name format.
	SchemaRuleFKNaming SQLReviewRuleType = "naming.index.fk"
	// SchemaRuleIDXNaming enforce the index name format.
	SchemaRuleIDXNaming SQLReviewRuleType = "naming.index.idx"
	// SchemaRuleAutoIncrementColumnNaming enforce the auto_increment column name format.
	SchemaRuleAutoIncrementColumnNaming SQLReviewRuleType = "naming.column.auto-increment"
	// SchemaRuleTableNameNoKeyword enforce the table name not to use keyword.
	SchemaRuleTableNameNoKeyword SQLReviewRuleType = "naming.table.no-keyword"
	// SchemaRuleIdentifierNoKeyword enforce the identifier not to use keyword.
	SchemaRuleIdentifierNoKeyword SQLReviewRuleType = "naming.identifier.no-keyword"
	// SchemaRuleIdentifierCase enforce the identifier case.
	SchemaRuleIdentifierCase SQLReviewRuleType = "naming.identifier.case"

	// SchemaRuleStatementNoSelectAll disallow 'SELECT *'.
	SchemaRuleStatementNoSelectAll SQLReviewRuleType = "statement.select.no-select-all"
	// SchemaRuleStatementRequireWhereForSelect require 'WHERE' clause for SELECT statements.
	SchemaRuleStatementRequireWhereForSelect SQLReviewRuleType = "statement.where.require.select"
	// SchemaRuleStatementRequireWhereForUpdateDelete require 'WHERE' clause for UPDATE and DELETE statements.
	SchemaRuleStatementRequireWhereForUpdateDelete SQLReviewRuleType = "statement.where.require.update-delete"
	// SchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	SchemaRuleStatementNoLeadingWildcardLike SQLReviewRuleType = "statement.where.no-leading-wildcard-like"
	// SchemaRuleStatementDisallowOnDelCascade disallows ON DELETE CASCADE clauses.
	SchemaRuleStatementDisallowOnDelCascade SQLReviewRuleType = "statement.disallow-on-del-cascade"
	// SchemaRuleStatementDisallowRemoveTblCascade disallows CASCADE when removing a table.
	SchemaRuleStatementDisallowRemoveTblCascade SQLReviewRuleType = "statement.disallow-rm-tbl-cascade"
	// SchemaRuleStatementDisallowCommit disallow using commit in the issue.
	SchemaRuleStatementDisallowCommit SQLReviewRuleType = "statement.disallow-commit"
	// SchemaRuleStatementDisallowLimit disallow the LIMIT clause in INSERT, DELETE and UPDATE statements.
	SchemaRuleStatementDisallowLimit SQLReviewRuleType = "statement.disallow-limit"
	// SchemaRuleStatementDisallowOrderBy disallow the ORDER BY clause in DELETE and UPDATE statements.
	SchemaRuleStatementDisallowOrderBy SQLReviewRuleType = "statement.disallow-order-by"
	// SchemaRuleStatementMergeAlterTable disallow redundant ALTER TABLE statements.
	SchemaRuleStatementMergeAlterTable SQLReviewRuleType = "statement.merge-alter-table"
	// SchemaRuleStatementInsertRowLimit enforce the insert row limit.
	SchemaRuleStatementInsertRowLimit SQLReviewRuleType = "statement.insert.row-limit"
	// SchemaRuleStatementInsertMustSpecifyColumn enforce the insert column specified.
	SchemaRuleStatementInsertMustSpecifyColumn SQLReviewRuleType = "statement.insert.must-specify-column"
	// SchemaRuleStatementInsertDisallowOrderByRand disallow the order by rand in the INSERT statement.
	SchemaRuleStatementInsertDisallowOrderByRand SQLReviewRuleType = "statement.insert.disallow-order-by-rand"
	// SchemaRuleStatementAffectedRowLimit enforce the UPDATE/DELETE affected row limit.
	SchemaRuleStatementAffectedRowLimit SQLReviewRuleType = "statement.affected-row-limit"
	// SchemaRuleStatementDMLDryRun dry run the dml.
	SchemaRuleStatementDMLDryRun SQLReviewRuleType = "statement.dml-dry-run"
	// SchemaRuleStatementDisallowAddColumnWithDefault disallow to add column with DEFAULT.
	SchemaRuleStatementDisallowAddColumnWithDefault = "statement.disallow-add-column-with-default"
	// SchemaRuleStatementAddCheckNotValid require add check constraints not valid.
	SchemaRuleStatementAddCheckNotValid = "statement.add-check-not-valid"
	// SchemaRuleStatementAddFKNotValid require add foreign key not valid.
	SchemaRuleStatementAddFKNotValid = "statement.add-foreign-key-not-valid"
	// SchemaRuleStatementDisallowAddNotNull disallow to add NOT NULL.
	SchemaRuleStatementDisallowAddNotNull = "statement.disallow-add-not-null"
	// SchemaRuleStatementDisallowAddColumn disallow to add column.
	SchemaRuleStatementSelectFullTableScan = "statement.select-full-table-scan"
	// SchemaRuleStatementCreateSpecifySchema disallow to create table without specifying schema.
	SchemaRuleStatementCreateSpecifySchema = "statement.create-specify-schema"
	// SchemaRuleStatementCheckSetRoleVariable require add a check for SET ROLE variable.
	SchemaRuleStatementCheckSetRoleVariable = "statement.check-set-role-variable"
	// SchemaRuleStatementDisallowUsingFilesort disallow using filesort in execution plan.
	SchemaRuleStatementDisallowUsingFilesort = "statement.disallow-using-filesort"
	// SchemaRuleStatementDisallowUsingTemporary disallow using temporary in execution plan.
	SchemaRuleStatementDisallowUsingTemporary = "statement.disallow-using-temporary"
	// SchemaRuleStatementWhereNoEqualNull check the WHERE clause no equal null.
	SchemaRuleStatementWhereNoEqualNull = "statement.where.no-equal-null"
	// SchemaRuleStatementWhereDisallowFunctionsAndCaculations disallow using function in WHERE clause.
	SchemaRuleStatementWhereDisallowFunctionsAndCaculations = "statement.where.disallow-functions-and-calculations"
	// SchemaRuleStatementQueryMinumumPlanLevel enforce the minimum plan level.
	SchemaRuleStatementQueryMinumumPlanLevel = "statement.query.minimum-plan-level"
	// SchemaRuleStatementWhereMaximumLogicalOperatorCount enforce the maximum logical operator count in WHERE clause.
	SchemaRuleStatementWhereMaximumLogicalOperatorCount = "statement.where.maximum-logical-operator-count"
	// SchemaRuleStatementMaximumLimitValue enforce the maximum limit value.
	SchemaRuleStatementMaximumLimitValue = "statement.maximum-limit-value"
	// SchemaRuleStatementMaximumJoinTableCount enforce the maximum join table count in the statement.
	SchemaRuleStatementMaximumJoinTableCount = "statement.maximum-join-table-count"
	// SchemaRuleStatementMaximumStatementsInTransaction enforce the maximum statements in transaction.
	SchemaRuleStatementMaximumStatementsInTransaction = "statement.maximum-statements-in-transaction"
	// SchemaRuleStatementJoinStrictColumnAttrs enforce the join strict column attributes.
	SchemaRuleStatementJoinStrictColumnAttrs = "statement.join-strict-column-attrs"
	// SchemaRuleStatementDisallowMixInDDL disallows DML statements in DDL statements.
	SchemaRuleStatementDisallowMixInDDL = "statement.disallow-mix-in-ddl"
	// SchemaRuleStatementDisallowMixInDML disallows DDL statements in DML statements.
	SchemaRuleStatementDisallowMixInDML = "statement.disallow-mix-in-dml"
	// SchemaRuleStatementPriorBackupCheck checks for prior backup.
	SchemaRuleStatementPriorBackupCheck = "statement.prior-backup-check"
	// SchemaRuleStatementNonTransactional checks for non-transactional statements.
	SchemaRuleStatementNonTransactional = "statement.non-transactional"
	// SchemaRuleStatementAddColumnWithoutPosition check no position in ADD COLUMN clause.
	SchemaRuleStatementAddColumnWithoutPosition = "statement.add-column-without-position"
	// SchemaRuleStatementDisallowOfflineDDL disallow offline ddl.
	SchemaRuleStatementDisallowOfflineDDL = "statement.disallow-offline-ddl"
	// SchemaRuleStatementDisallowCrossDBQueries disallow cross database queries.
	SchemaRuleStatementDisallowCrossDBQueries = "statement.disallow-cross-db-queries"
	// SchemaRuleStatementMaxExecutionTime enforce the maximum execution time.
	SchemaRuleStatementMaxExecutionTime = "statement.max-execution-time"
	// SchemaRuleStatementRequireAlgorithmOption require set ALGORITHM option in ALTER TABLE statement.
	SchemaRuleStatementRequireAlgorithmOption = "statement.require-algorithm-option"
	// SchemaRuleStatementRequireLockOption require set LOCK option in ALTER TABLE statement.
	SchemaRuleStatementRequireLockOption = "statement.require-lock-option"
	// SchemaRuleStatementObjectOwnerCheck checks the object owner for the statement.
	SchemaRuleStatementObjectOwnerCheck = "statement.object-owner-check"
	// SchemaRuleTableRequirePK require the table to have a primary key.
	SchemaRuleTableRequirePK SQLReviewRuleType = "table.require-pk"
	// SchemaRuleTableNoFK require the table disallow the foreign key.
	SchemaRuleTableNoFK SQLReviewRuleType = "table.no-foreign-key"
	// SchemaRuleTableDropNamingConvention require only the table following the naming convention can be deleted.
	SchemaRuleTableDropNamingConvention SQLReviewRuleType = "table.drop-naming-convention"
	// SchemaRuleTableCommentConvention enforce the table comment convention.
	SchemaRuleTableCommentConvention SQLReviewRuleType = "table.comment"
	// SchemaRuleTableDisallowPartition disallow the table partition.
	SchemaRuleTableDisallowPartition SQLReviewRuleType = "table.disallow-partition"
	// SchemaRuleTableDisallowTrigger disallow the table trigger.
	SchemaRuleTableDisallowTrigger SQLReviewRuleType = "table.disallow-trigger"
	// SchemaRuleTableNoDuplicateIndex require the table no duplicate index.
	SchemaRuleTableNoDuplicateIndex SQLReviewRuleType = "table.no-duplicate-index"
	// SchemaRuleTableTextFieldsTotalLength enforce the total length of text fields.
	SchemaRuleTableTextFieldsTotalLength SQLReviewRuleType = "table.text-fields-total-length"
	// SchemaRuleTableDisallowSetCharset disallow set table charset.
	SchemaRuleTableDisallowSetCharset SQLReviewRuleType = "table.disallow-set-charset"
	// SchemaRuleTableDisallowDDL disallow executing DDL for specific tables.
	SchemaRuleTableDisallowDDL SQLReviewRuleType = "table.disallow-ddl"
	// SchemaRuleTableDisallowDML disallow executing DML on specific tables.
	SchemaRuleTableDisallowDML SQLReviewRuleType = "table.disallow-dml"
	// SchemaRuleTableLimitSize  restrict access to tables based on size.
	SchemaRuleTableLimitSize SQLReviewRuleType = "table.limit-size"
	// SchemaRuleTableRequireCharset enforce the table charset.
	SchemaRuleTableRequireCharset SQLReviewRuleType = "table.require-charset"
	// SchemaRuleTableRequireCollation enforce the table collation.
	SchemaRuleTableRequireCollation SQLReviewRuleType = "table.require-collation"
	// SchemaRuleRequiredColumn enforce the required columns in each table.
	SchemaRuleRequiredColumn SQLReviewRuleType = "column.required"
	// SchemaRuleColumnNotNull enforce the columns cannot have NULL value.
	SchemaRuleColumnNotNull SQLReviewRuleType = "column.no-null"
	// SchemaRuleColumnDisallowChangeType disallow change column type.
	SchemaRuleColumnDisallowChangeType SQLReviewRuleType = "column.disallow-change-type"
	// SchemaRuleColumnSetDefaultForNotNull require the not null column to set default value.
	SchemaRuleColumnSetDefaultForNotNull SQLReviewRuleType = "column.set-default-for-not-null"
	// SchemaRuleColumnDisallowChange disallow CHANGE COLUMN statement.
	SchemaRuleColumnDisallowChange SQLReviewRuleType = "column.disallow-change"
	// SchemaRuleColumnDisallowChangingOrder disallow changing column order.
	SchemaRuleColumnDisallowChangingOrder SQLReviewRuleType = "column.disallow-changing-order"
	// SchemaRuleColumnDisallowDrop disallow drop column.
	SchemaRuleColumnDisallowDrop SQLReviewRuleType = "column.disallow-drop"
	// SchemaRuleColumnDisallowDropInIndex disallow drop index column.
	SchemaRuleColumnDisallowDropInIndex SQLReviewRuleType = "column.disallow-drop-in-index"
	// SchemaRuleColumnCommentConvention enforce the column comment convention.
	SchemaRuleColumnCommentConvention SQLReviewRuleType = "column.comment"
	// SchemaRuleColumnAutoIncrementMustInteger require the auto-increment column to be integer.
	SchemaRuleColumnAutoIncrementMustInteger SQLReviewRuleType = "column.auto-increment-must-integer"
	// SchemaRuleColumnTypeDisallowList enforce the column type disallow list.
	SchemaRuleColumnTypeDisallowList SQLReviewRuleType = "column.type-disallow-list"
	// SchemaRuleColumnDisallowSetCharset disallow set column charset.
	SchemaRuleColumnDisallowSetCharset SQLReviewRuleType = "column.disallow-set-charset"
	// SchemaRuleColumnMaximumCharacterLength enforce the maximum character length.
	SchemaRuleColumnMaximumCharacterLength SQLReviewRuleType = "column.maximum-character-length"
	// SchemaRuleColumnMaximumVarcharLength enforce the maximum varchar length.
	SchemaRuleColumnMaximumVarcharLength SQLReviewRuleType = "column.maximum-varchar-length"
	// SchemaRuleColumnAutoIncrementInitialValue enforce the initial auto-increment value.
	SchemaRuleColumnAutoIncrementInitialValue SQLReviewRuleType = "column.auto-increment-initial-value"
	// SchemaRuleColumnAutoIncrementMustUnsigned enforce the auto-increment column to be unsigned.
	SchemaRuleColumnAutoIncrementMustUnsigned SQLReviewRuleType = "column.auto-increment-must-unsigned"
	// SchemaRuleCurrentTimeColumnCountLimit enforce the current column count limit.
	SchemaRuleCurrentTimeColumnCountLimit SQLReviewRuleType = "column.current-time-count-limit"
	// SchemaRuleColumnRequireDefault enforce the column default.
	SchemaRuleColumnRequireDefault SQLReviewRuleType = "column.require-default"
	// SchemaRuleColumnDefaultDisallowVolatile enforce the column default disallow volatile.
	SchemaRuleColumnDefaultDisallowVolatile SQLReviewRuleType = "column.default-disallow-volatile"
	// SchemaRuleAddNotNullColumnRequireDefault enforce the adding not null column requires default.
	SchemaRuleAddNotNullColumnRequireDefault SQLReviewRuleType = "column.add-not-null-require-default"
	// SchemaRuleColumnRequireCharset enforce the column require charset.
	SchemaRuleColumnRequireCharset SQLReviewRuleType = "column.require-charset"
	// SchemaRuleColumnRequireCollation enforce the column require collation.
	SchemaRuleColumnRequireCollation SQLReviewRuleType = "column.require-collation"

	// SchemaRuleSchemaBackwardCompatibility enforce the MySQL and TiDB support check whether the schema change is backward compatible.
	SchemaRuleSchemaBackwardCompatibility SQLReviewRuleType = "schema.backward-compatibility"

	// SchemaRuleDropEmptyDatabase enforce the MySQL and TiDB support check if the database is empty before users drop it.
	SchemaRuleDropEmptyDatabase SQLReviewRuleType = "database.drop-empty-database"

	// SchemaRuleIndexNoDuplicateColumn require the index no duplicate column.
	SchemaRuleIndexNoDuplicateColumn SQLReviewRuleType = "index.no-duplicate-column"
	// SchemaRuleIndexKeyNumberLimit enforce the index key number limit.
	SchemaRuleIndexKeyNumberLimit SQLReviewRuleType = "index.key-number-limit"
	// SchemaRuleIndexPKTypeLimit enforce the type restriction of columns in primary key.
	SchemaRuleIndexPKTypeLimit SQLReviewRuleType = "index.pk-type-limit"
	// SchemaRuleIndexTypeNoBlob enforce the type restriction of columns in index.
	SchemaRuleIndexTypeNoBlob SQLReviewRuleType = "index.type-no-blob"
	// SchemaRuleIndexTotalNumberLimit enforce the index total number limit.
	SchemaRuleIndexTotalNumberLimit SQLReviewRuleType = "index.total-number-limit"
	// SchemaRuleIndexPrimaryKeyTypeAllowlist enforce the primary key type allowlist.
	SchemaRuleIndexPrimaryKeyTypeAllowlist SQLReviewRuleType = "index.primary-key-type-allowlist"
	// SchemaRuleCreateIndexConcurrently require creating indexes concurrently.
	SchemaRuleCreateIndexConcurrently SQLReviewRuleType = "index.create-concurrently"
	// SchemaRuleIndexTypeAllowList enforce the index type allowlist.
	SchemaRuleIndexTypeAllowList SQLReviewRuleType = "index.type-allow-list"
	// SchemaRuleIndexNotRedundant prohibits createing redundant indices.
	SchemaRuleIndexNotRedundant SQLReviewRuleType = "index.not-redundant"

	// SchemaRuleCharsetAllowlist enforce the charset allowlist.
	SchemaRuleCharsetAllowlist SQLReviewRuleType = "system.charset.allowlist"
	// SchemaRuleCollationAllowlist enforce the collation allowlist.
	SchemaRuleCollationAllowlist SQLReviewRuleType = "system.collation.allowlist"
	// SchemaRuleCommentLength limit comment length.
	SchemaRuleCommentLength SQLReviewRuleType = "system.comment.length"
	// SchemaRuleProcedureDisallowCreate disallow create procedure.
	SchemaRuleProcedureDisallowCreate SQLReviewRuleType = "system.procedure.disallow-create"
	// SchemaRuleEventDisallowCreate disallow create event.
	SchemaRuleEventDisallowCreate SQLReviewRuleType = "system.event.disallow-create"
	// SchemaRuleViewDisallowCreate disallow create view.
	SchemaRuleViewDisallowCreate SQLReviewRuleType = "system.view.disallow-create"
	// SchemaRuleFunctionDisallowCreate disallow create function.
	SchemaRuleFunctionDisallowCreate SQLReviewRuleType = "system.function.disallow-create"
	// SchemaRuleFunctionDisallowList enforce the disallowed function list.
	SchemaRuleFunctionDisallowList SQLReviewRuleType = "system.function.disallowed-list"

	// SchemaRuleOnlineMigration advises using online migration to migrate large tables.
	SchemaRuleOnlineMigration SQLReviewRuleType = "advice.online-migration"

	// TableNameTemplateToken is the token for table name.
	TableNameTemplateToken = "{{table}}"
	// ColumnListTemplateToken is the token for column name list.
	ColumnListTemplateToken = "{{column_list}}"
	// ReferencingTableNameTemplateToken is the token for referencing table name.
	ReferencingTableNameTemplateToken = "{{referencing_table}}"
	// ReferencingColumnNameTemplateToken is the token for referencing column name.
	ReferencingColumnNameTemplateToken = "{{referencing_column}}"
	// ReferencedTableNameTemplateToken is the token for referenced table name.
	ReferencedTableNameTemplateToken = "{{referenced_table}}"
	// ReferencedColumnNameTemplateToken is the token for referenced column name.
	ReferencedColumnNameTemplateToken = "{{referenced_column}}"

	// defaultNameLengthLimit is the default length limit for naming rules.
	// PostgreSQL has it's own naming length limit, will auto slice the name to make sure its length <= 63
	// https://www.postgresql.org/docs/current/limits.html.
	// While MySQL does not enforce the limit, thus we use PostgreSQL's 63 as the default limit.
	defaultNameLengthLimit = 63
)

var (
	// TemplateNamingTokens is the mapping for rule type to template token.
	TemplateNamingTokens = map[SQLReviewRuleType]map[string]bool{
		SchemaRuleIDXNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		SchemaRulePKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		SchemaRuleUKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		SchemaRuleFKNaming: {
			ReferencingTableNameTemplateToken:  true,
			ReferencingColumnNameTemplateToken: true,
			ReferencedTableNameTemplateToken:   true,
			ReferencedColumnNameTemplateToken:  true,
		},
	}
)

// NamingRulePayload is the payload for naming rule.
type NamingRulePayload struct {
	MaxLength int    `json:"maxLength"`
	Format    string `json:"format"`
}

// StringArrayTypeRulePayload is the payload for rules with string array value.
type StringArrayTypeRulePayload struct {
	List []string `json:"list"`
}

// RequiredColumnRulePayload is the payload for required column rule.
type RequiredColumnRulePayload struct {
	ColumnList []string `json:"columnList"`
}

// CommentConventionRulePayload is the payload for comment convention rule.
type CommentConventionRulePayload struct {
	Required               bool `json:"required"`
	RequiredClassification bool `json:"requiredClassification"`
	MaxLength              int  `json:"maxLength"`
}

// NumberTypeRulePayload is the number type payload.
type NumberTypeRulePayload struct {
	Number int `json:"number"`
}

// StringTypeRulePayload is the string type payload.
type StringTypeRulePayload struct {
	String string `json:"string"`
}

// NamingCaseRulePayload is the payload for naming case rule.
type NamingCaseRulePayload struct {
	// Upper is true means the case should be upper case, otherwise lower case.
	Upper bool `json:"upper"`
}

// UnmarshalNamingRulePayloadAsRegexp will unmarshal payload to NamingRulePayload and compile it as regular expression.
func UnmarshalNamingRulePayloadAsRegexp(payload string) (*regexp.Regexp, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	format, err := regexp.Compile(nr.Format)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to compile regular expression \"%s\"", nr.Format)
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = defaultNameLengthLimit
	}

	return format, maxLength, nil
}

// UnmarshalNamingRulePayloadAsTemplate will unmarshal payload to NamingRulePayload and extract all the template keys.
// For example, "hard_code_{{table}}_{{column}}_end" will return
// "hard_code_{{table}}_{{column}}_end", ["{{table}}", "{{column}}"].
func UnmarshalNamingRulePayloadAsTemplate(ruleType SQLReviewRuleType, payload string) (string, []string, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return "", nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	template := nr.Format
	keys, _ := parseTemplateTokens(template)

	for _, key := range keys {
		if _, ok := TemplateNamingTokens[ruleType][key]; !ok {
			return "", nil, 0, errors.Errorf("invalid template %s for rule %s", key, ruleType)
		}
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = defaultNameLengthLimit
	}

	return template, keys, maxLength, nil
}

// parseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters such as "$" in the delimiters. The caller should escape the delimiters in such cases.
func parseTemplateTokens(template string) ([]string, []string) {
	r := regexp.MustCompile(`{{[^{}]+}}`)
	tokens := r.FindAllString(template, -1)
	if len(tokens) > 0 {
		split := r.Split(template, -1)
		var delimiters []string
		for _, s := range split {
			if s != "" {
				delimiters = append(delimiters, s)
			}
		}
		return tokens, delimiters
	}
	return nil, nil
}

// UnmarshalRequiredColumnList will unmarshal payload and parse the required column list.
func UnmarshalRequiredColumnList(payload string) ([]string, error) {
	stringArrayRulePayload, err := UnmarshalStringArrayTypeRulePayload(payload)
	if err != nil {
		return nil, err
	}
	if len(stringArrayRulePayload.List) != 0 {
		return stringArrayRulePayload.List, nil
	}

	// The RequiredColumnRulePayload is deprecated.
	// Just keep it to compatible with old data, and we can remove this later.
	columnRulePayload, err := unmarshalRequiredColumnRulePayload(payload)
	if err != nil {
		return nil, err
	}

	return columnRulePayload.ColumnList, nil
}

// unmarshalRequiredColumnRulePayload will unmarshal payload to RequiredColumnRulePayload.
func unmarshalRequiredColumnRulePayload(payload string) (*RequiredColumnRulePayload, error) {
	var rcr RequiredColumnRulePayload
	if err := json.Unmarshal([]byte(payload), &rcr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal required column rule payload %q", payload)
	}
	if len(rcr.ColumnList) == 0 {
		return nil, errors.Errorf("invalid required column rule payload, column list cannot be empty")
	}
	return &rcr, nil
}

// UnmarshalCommentConventionRulePayload will unmarshal payload to CommentConventionRulePayload.
func UnmarshalCommentConventionRulePayload(payload string) (*CommentConventionRulePayload, error) {
	var ccr CommentConventionRulePayload
	if err := json.Unmarshal([]byte(payload), &ccr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal comment convention rule payload %q", payload)
	}
	return &ccr, nil
}

// UnmarshalNumberTypeRulePayload will unmarshal payload to NumberTypeRulePayload.
func UnmarshalNumberTypeRulePayload(payload string) (*NumberTypeRulePayload, error) {
	var nlr NumberTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &nlr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal number type rule payload %q", payload)
	}
	return &nlr, nil
}

// UnmarshalStringTypeRulePayload will unmarshal payload to StringTypeRulePayload.
func UnmarshalStringTypeRulePayload(payload string) (*StringTypeRulePayload, error) {
	var slr StringTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &slr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string type rule payload %q", payload)
	}
	return &slr, nil
}

// UnmarshalStringArrayTypeRulePayload will unmarshal payload to StringArrayTypeRulePayload.
func UnmarshalStringArrayTypeRulePayload(payload string) (*StringArrayTypeRulePayload, error) {
	var trr StringArrayTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &trr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string array rule payload %q", payload)
	}
	return &trr, nil
}

// UnmarshalNamingCaseRulePayload will unmarshal payload to NamingCaseRulePayload.
func UnmarshalNamingCaseRulePayload(payload string) (*NamingCaseRulePayload, error) {
	var ncr NamingCaseRulePayload
	if err := json.Unmarshal([]byte(payload), &ncr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal naming case rule payload %q", payload)
	}
	return &ncr, nil
}

// Catalog is the service for catalog.
type catalogInterface interface {
	GetFinder() *catalog.Finder
}

// SQLReviewCheckContext is the context for SQL review check.
type SQLReviewCheckContext struct {
	Charset               string
	Collation             string
	ChangeType            storepb.PlanCheckRunConfig_ChangeDatabaseType
	DBSchema              *storepb.DatabaseSchemaMetadata
	DbType                storepb.Engine
	Catalog               catalogInterface
	Driver                *sql.DB
	Context               context.Context
	PreUpdateBackupDetail *storepb.PreUpdateBackupDetail
	ClassificationConfig  *storepb.DataClassificationSetting_DataClassificationConfig
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string

	// Snowflake specific fields
	CurrentDatabase string

	// Used for test only.
	NoAppendBuiltin bool

	// UsePostgresDatabaseOwner is true if the advisor should use the database owner as default role.
	UsePostgresDatabaseOwner bool
}

// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(
	sm *sheet.Manager,
	statements string,
	ruleList []*storepb.SQLReviewRule,
	checkContext SQLReviewCheckContext,
) ([]*storepb.Advice, error) {
	asts, parseResult := sm.GetASTsForChecks(checkContext.DbType, statements)

	builtinOnly := len(ruleList) == 0

	if !checkContext.NoAppendBuiltin {
		// Append builtin rules to the rule list.
		ruleList = append(ruleList, GetBuiltinRules(checkContext.DbType)...)
	}

	if asts == nil || len(ruleList) == 0 {
		return parseResult, nil
	}

	finder := checkContext.Catalog.GetFinder()
	if !builtinOnly {
		switch checkContext.DbType {
		case storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
			if err := finder.WalkThrough(asts); err != nil {
				return convertWalkThroughErrorToAdvice(checkContext, err)
			}
		}
	}

	var errorAdvices, warningAdvices []*storepb.Advice
	for _, rule := range ruleList {
		if rule.Engine != storepb.Engine_ENGINE_UNSPECIFIED && rule.Engine != checkContext.DbType {
			continue
		}
		if rule.Level == storepb.SQLReviewRuleLevel_DISABLED {
			continue
		}

		ruleType := SQLReviewRuleType(rule.Type)
		if !isRuleAllowed(ruleType, checkContext.ChangeType) {
			continue
		}

		advisorType, err := getAdvisorTypeByRule(ruleType, checkContext.DbType)
		if err != nil {
			if rule.Engine != storepb.Engine_ENGINE_UNSPECIFIED {
				slog.Warn("not supported rule", "rule type", rule.Type, "engine", rule.Engine.String(), log.BBError(err))
			}
			continue
		}

		adviceList, err := Check(
			checkContext.DbType,
			advisorType,
			Context{
				Charset:                  checkContext.Charset,
				Collation:                checkContext.Collation,
				DBSchema:                 checkContext.DBSchema,
				ChangeType:               checkContext.ChangeType,
				PreUpdateBackupDetail:    checkContext.PreUpdateBackupDetail,
				AST:                      asts,
				Statements:               statements,
				Rule:                     rule,
				Catalog:                  finder,
				Driver:                   checkContext.Driver,
				Context:                  checkContext.Context,
				CurrentDatabase:          checkContext.CurrentDatabase,
				ClassificationConfig:     checkContext.ClassificationConfig,
				UsePostgresDatabaseOwner: checkContext.UsePostgresDatabaseOwner,
				ListDatabaseNamesFunc:    checkContext.ListDatabaseNamesFunc,
				InstanceID:               checkContext.InstanceID,
			},
			statements,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check statement")
		}

		for _, advice := range adviceList {
			switch advice.Status {
			case storepb.Advice_ERROR:
				if len(errorAdvices) < common.MaximumAdvicePerStatus {
					errorAdvices = append(errorAdvices, advice)
				}
			case storepb.Advice_WARNING:
				if len(warningAdvices) < common.MaximumAdvicePerStatus {
					warningAdvices = append(warningAdvices, advice)
				}
			default:
			}
		}
		// Skip remaining rules if we have enough error and warning advices.
		if len(errorAdvices) >= common.MaximumAdvicePerStatus && len(warningAdvices) >= common.MaximumAdvicePerStatus {
			break
		}
	}

	var advices []*storepb.Advice
	advices = append(advices, errorAdvices...)
	advices = append(advices, warningAdvices...)
	return advices, nil
}

func convertWalkThroughErrorToAdvice(checkContext SQLReviewCheckContext, err error) ([]*storepb.Advice, error) {
	walkThroughError, ok := err.(*catalog.WalkThroughError)
	if !ok {
		return nil, err
	}

	var res []*storepb.Advice
	switch walkThroughError.Type {
	case catalog.ErrorTypeUnsupported:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    Unsupported.Int32(),
			Title:   walkThroughError.Content,
			Content: "",
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeParseError:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    StatementSyntaxError.Int32(),
			Title:   SyntaxErrorTitle,
			Content: walkThroughError.Content,
		})
	case catalog.ErrorTypeDeparseError:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    Internal.Int32(),
			Title:   "Internal error for walk-through",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeAccessOtherDatabase:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    NotCurrentDatabase.Int32(),
			Title:   "Access other database",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeDatabaseIsDeleted:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    DatabaseIsDeleted.Int32(),
			Title:   "Access deleted database",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeTableExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    TableExists.Int32(),
			Title:   "Table already exists",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeTableNotExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    TableNotExists.Int32(),
			Title:   "Table does not exist",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeColumnExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    ColumnExists.Int32(),
			Title:   "Column already exists",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeColumnNotExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    ColumnNotExists.Int32(),
			Title:   "Column does not exist",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeDropAllColumns:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    DropAllColumns.Int32(),
			Title:   "Drop all columns",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypePrimaryKeyExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    PrimaryKeyExists.Int32(),
			Title:   "Primary key exists",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeIndexExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    IndexExists.Int32(),
			Title:   "Index exists",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeIndexEmptyKeys:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    IndexEmptyKeys.Int32(),
			Title:   "Index empty keys",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypePrimaryKeyNotExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    PrimaryKeyNotExists.Int32(),
			Title:   "Primary key does not exist",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeIndexNotExists:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    IndexNotExists.Int32(),
			Title:   "Index does not exist",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeIncorrectIndexName:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    IncorrectIndexName.Int32(),
			Title:   "Incorrect index name",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeSpatialIndexKeyNullable:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    SpatialIndexKeyNullable.Int32(),
			Title:   "Spatial index key must be NOT NULL",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	case catalog.ErrorTypeColumnIsReferencedByView:
		details := ""
		if checkContext.DbType == storepb.Engine_POSTGRES {
			list, yes := walkThroughError.Payload.([]string)
			if !yes {
				return nil, errors.Errorf("invalid payload for ColumnIsReferencedByView, expect []string but found %T", walkThroughError.Payload)
			}
			if definition, err := getViewDefinition(checkContext, list); err != nil {
				slog.Warn("failed to get view definition", log.BBError(err))
			} else {
				details = definition
			}
		}

		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    ColumnIsReferencedByView.Int32(),
			Title:   "Column is referenced by view",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
			Detail: details,
		})
	case catalog.ErrorTypeTableIsReferencedByView:
		details := ""
		if checkContext.DbType == storepb.Engine_POSTGRES {
			list, yes := walkThroughError.Payload.([]string)
			if !yes {
				return nil, errors.Errorf("invalid payload for TableIsReferencedByView, expect []string but found %T", walkThroughError.Payload)
			}
			if definition, err := getViewDefinition(checkContext, list); err != nil {
				slog.Warn("failed to get view definition", log.BBError(err))
			} else {
				details = definition
			}
		}

		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    TableIsReferencedByView.Int32(),
			Title:   "Table is referenced by view",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
			Detail: details,
		})
	case catalog.ErrorTypeInvalidColumnTypeForDefaultValue:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    InvalidColumnDefault.Int32(),
			Title:   "Invalid column default value",
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	default:
		res = append(res, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    Internal.Int32(),
			Title:   fmt.Sprintf("Failed to walk-through with code %d", walkThroughError.Type),
			Content: walkThroughError.Content,
			StartPosition: &storepb.Position{
				Line: int32(walkThroughError.Line),
			},
		})
	}

	return res, nil
}

func getViewDefinition(checkContext SQLReviewCheckContext, viewList []string) (string, error) {
	var buf bytes.Buffer
	sql := fmt.Sprintf(`
		WITH view_list(view_name) AS (
			VALUES ('%s')
		)
		SELECT view_name, pg_get_viewdef(view_name) AS view_definition
		FROM view_list;
	`, strings.Join(viewList, "'),('"))

	rows, err := checkContext.Driver.QueryContext(checkContext.Context, sql)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var viewName, viewDefinition string
		if err := rows.Scan(&viewName, &viewDefinition); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("The definition of view "); err != nil {
			return "", err
		}
		if _, err := buf.WriteString(viewName); err != nil {
			return "", err
		}
		if _, err := buf.WriteString(" is \n"); err != nil {
			return "", err
		}
		if _, err := buf.WriteString(viewDefinition); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n\n"); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getAdvisorTypeByRule(ruleType SQLReviewRuleType, engine storepb.Engine) (Type, error) {
	switch ruleType {
	case SchemaRuleStatementRequireWhereForSelect:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLWhereRequirementForSelect, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLWhereRequirementForSelect, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleWhereRequirementForSelect, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeWhereRequirementForSelect, nil
		case storepb.Engine_MSSQL:
			return MSSQLWhereRequirementForSelect, nil
		}
	case SchemaRuleStatementRequireWhereForUpdateDelete:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLWhereRequirementForUpdateDelete, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLWhereRequirementForUpdateDelete, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleWhereRequirementForUpdateDelete, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeWhereRequirementForUpdateDelete, nil
		case storepb.Engine_MSSQL:
			return MSSQLWhereRequirementForUpdateDelete, nil
		}
	case SchemaRuleStatementNoLeadingWildcardLike:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNoLeadingWildcardLike, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNoLeadingWildcardLike, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleNoLeadingWildcardLike, nil
		}
	case SchemaRuleStatementNoSelectAll:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNoSelectAll, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNoSelectAll, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleNoSelectAll, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeNoSelectAll, nil
		case storepb.Engine_MSSQL:
			return MSSQLNoSelectAll, nil
		}
	case SchemaRuleSchemaBackwardCompatibility:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLMigrationCompatibility, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLMigrationCompatibility, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeMigrationCompatibility, nil
		case storepb.Engine_MSSQL:
			return MSSQLMigrationCompatibility, nil
		}
	case SchemaRuleFullyQualifiedObjectName:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLNamingFullyQualifiedObjectName, nil
		}
	case SchemaRuleTableNaming:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNamingTableConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNamingTableConvention, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleNamingTableConvention, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeNamingTableConvention, nil
		case storepb.Engine_MSSQL:
			return MSSQLNamingTableConvention, nil
		}
	case SchemaRuleIDXNaming:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNamingIndexConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNamingIndexConvention, nil
		}
	case SchemaRulePKNaming:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLNamingPKConvention, nil
		}
	case SchemaRuleUKNaming:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNamingUKConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNamingUKConvention, nil
		}
	case SchemaRuleFKNaming:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNamingFKConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNamingFKConvention, nil
		}
	case SchemaRuleColumnNaming:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNamingColumnConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLNamingColumnConvention, nil
		}
	case SchemaRuleAutoIncrementColumnNaming:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLNamingAutoIncrementColumnConvention, nil
		}
	case SchemaRuleTableNameNoKeyword:
		switch engine {
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleTableNamingNoKeyword, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeTableNamingNoKeyword, nil
		case storepb.Engine_MSSQL:
			return MSSQLTableNamingNoKeyword, nil
		}
	case SchemaRuleIdentifierNoKeyword:
		switch engine {
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleIdentifierNamingNoKeyword, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeIdentifierNamingNoKeyword, nil
		case storepb.Engine_MSSQL:
			return MSSQLIdentifierNamingNoKeyword, nil
		case storepb.Engine_MYSQL:
			return MySQLIdentifierNamingNoKeyword, nil
		}
	case SchemaRuleIdentifierCase:
		switch engine {
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleIdentifierCase, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeIdentifierCase, nil
		}
	case SchemaRuleRequiredColumn:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnRequirement, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLColumnRequirement, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleColumnRequirement, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeColumnRequirement, nil
		case storepb.Engine_MSSQL:
			return MSSQLColumnRequirement, nil
		}
	case SchemaRuleColumnNotNull:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnNoNull, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLColumnNoNull, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleColumnNoNull, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeColumnNoNull, nil
		case storepb.Engine_MSSQL:
			return MSSQLColumnNoNull, nil
		}
	case SchemaRuleColumnDisallowChangeType:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnDisallowChangingType, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLColumnDisallowChangingType, nil
		}
	case SchemaRuleColumnSetDefaultForNotNull:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnSetDefaultForNotNull, nil
		}
	case SchemaRuleColumnDisallowChange:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnDisallowChanging, nil
		}
	case SchemaRuleColumnDisallowChangingOrder:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnDisallowChangingOrder, nil
		}
	case SchemaRuleColumnDisallowDrop:
		if engine == storepb.Engine_OCEANBASE {
			return MySQLColumnDisallowDrop, nil
		}
	case SchemaRuleColumnDisallowDropInIndex:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnDisallowDropInIndex, nil
		}
	case SchemaRuleColumnCommentConvention:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnCommentConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLColumnCommentConvention, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleColumnCommentConvention, nil
		}
	case SchemaRuleColumnAutoIncrementMustInteger:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLAutoIncrementColumnMustInteger, nil
		}
	case SchemaRuleColumnTypeDisallowList:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnTypeDisallowList, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLColumnTypeDisallowList, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleColumnTypeDisallowList, nil
		case storepb.Engine_MSSQL:
			return MSSQLColumnTypeDisallowList, nil
		}
	case SchemaRuleColumnDisallowSetCharset:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLDisallowSetColumnCharset, nil
		}
	case SchemaRuleColumnMaximumCharacterLength:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnMaximumCharacterLength, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLColumnMaximumCharacterLength, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleColumnMaximumCharacterLength, nil
		}
	case SchemaRuleColumnMaximumVarcharLength:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLColumnMaximumVarcharLength, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleColumnMaximumVarcharLength, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeColumnMaximumVarcharLength, nil
		case storepb.Engine_MSSQL:
			return MSSQLColumnMaximumVarcharLength, nil
		}
	case SchemaRuleColumnAutoIncrementInitialValue:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLAutoIncrementColumnInitialValue, nil
		}
	case SchemaRuleColumnAutoIncrementMustUnsigned:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLAutoIncrementColumnMustUnsigned, nil
		}
	case SchemaRuleCurrentTimeColumnCountLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLCurrentTimeColumnCountLimit, nil
		}
	case SchemaRuleColumnRequireDefault:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLRequireColumnDefault, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLRequireColumnDefault, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleRequireColumnDefault, nil
		}
	case SchemaRuleColumnDefaultDisallowVolatile:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLColumnDefaultDisallowVolatile, nil
		}
	case SchemaRuleAddNotNullColumnRequireDefault:
		if engine == storepb.Engine_ORACLE {
			return OracleAddNotNullColumnRequireDefault, nil
		}
	case SchemaRuleColumnRequireCharset:
		if engine == storepb.Engine_MYSQL {
			return MySQLColumnRequireCharset, nil
		}
	case SchemaRuleColumnRequireCollation:
		if engine == storepb.Engine_MYSQL {
			return MySQLColumnRequireCollation, nil
		}
	case SchemaRuleTableRequirePK:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLTableRequirePK, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLTableRequirePK, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleTableRequirePK, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeTableRequirePK, nil
		case storepb.Engine_MSSQL:
			return MSSQLTableRequirePK, nil
		}
	case SchemaRuleTableNoFK:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLTableNoFK, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLTableNoFK, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleTableNoFK, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeTableNoFK, nil
		case storepb.Engine_MSSQL:
			return MSSQLTableNoFK, nil
		}
	case SchemaRuleTableDropNamingConvention:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLTableDropNamingConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLTableDropNamingConvention, nil
		case storepb.Engine_SNOWFLAKE:
			return SnowflakeTableDropNamingConvention, nil
		case storepb.Engine_MSSQL:
			return MSSQLTableDropNamingConvention, nil
		}
	case SchemaRuleTableCommentConvention:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLTableCommentConvention, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLTableCommentConvention, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleTableCommentConvention, nil
		}
	case SchemaRuleTableDisallowPartition:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLTableDisallowPartition, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLTableDisallowPartition, nil
		}
	case SchemaRuleTableDisallowTrigger:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableDisallowTrigger, nil
		}
	case SchemaRuleTableNoDuplicateIndex:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableNoDuplicateIndex, nil
		}
	case SchemaRuleTableTextFieldsTotalLength:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableTextFieldsTotalLength, nil
		}
	case SchemaRuleTableDisallowSetCharset:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableDisallowSetCharset, nil
		}
	case SchemaRuleTableDisallowDDL:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableDisallowDDL, nil
		} else if engine == storepb.Engine_MSSQL {
			return MSSQLTableDisallowDDL, nil
		}
	case SchemaRuleTableDisallowDML:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableDisallowDML, nil
		} else if engine == storepb.Engine_MSSQL {
			return MSSQLTableDisallowDML, nil
		}
	case SchemaRuleTableLimitSize:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableLimitSize, nil
		}
	case SchemaRuleTableRequireCharset:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableRequireCharset, nil
		}
	case SchemaRuleTableRequireCollation:
		if engine == storepb.Engine_MYSQL {
			return MySQLTableRequireCollation, nil
		}
	case SchemaRuleMySQLEngine:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
			return MySQLUseInnoDB, nil
		}
	case SchemaRuleDropEmptyDatabase:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLDatabaseAllowDropIfEmpty, nil
		}
	case SchemaRuleIndexNoDuplicateColumn:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLIndexNoDuplicateColumn, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLIndexNoDuplicateColumn, nil
		}
	case SchemaRuleIndexKeyNumberLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLIndexKeyNumberLimit, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLIndexKeyNumberLimit, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleIndexKeyNumberLimit, nil
		}
	case SchemaRuleIndexTotalNumberLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLIndexTotalNumberLimit, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLIndexTotalNumberLimit, nil
		}
	case SchemaRuleIndexNotRedundant:
		if engine == storepb.Engine_MSSQL {
			return MSSQLIndexNotRedundant, nil
		}
	case SchemaRuleStatementDisallowRemoveTblCascade:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLStatementDisallowRemoveTblCascade, nil
		}
	case SchemaRuleStatementDisallowOnDelCascade:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLStatementDisallowOnDelCascade, nil
		}
	case SchemaRuleStatementDisallowCommit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLStatementDisallowCommit, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLStatementDisallowCommit, nil
		}
	case SchemaRuleStatementDisallowUsingFilesort:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementDisallowUsingFilesort, nil
		}
	case SchemaRuleStatementDisallowUsingTemporary:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementDisallowUsingTemporary, nil
		}
	case SchemaRuleStatementDisallowMixInDDL:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB:
			return MySQLStatementDisallowMixInDDL, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLStatementDisallowMixInDDL, nil
		case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
			return OracleStatementDisallowMixInDDL, nil
		case storepb.Engine_MSSQL:
			return MSSQLStatementDisallowMixInDDL, nil
		}
	case SchemaRuleStatementDisallowMixInDML:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB:
			return MySQLStatementDisallowMixInDML, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLStatementDisallowMixInDML, nil
		case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
			return OracleStatementDisallowMixInDML, nil
		case storepb.Engine_MSSQL:
			return MSSQLStatementDisallowMixInDML, nil
		}
	case SchemaRuleStatementAddColumnWithoutPosition:
		if engine == storepb.Engine_OCEANBASE {
			return MySQLStatementAddColumnWithoutPosition, nil
		}
	case SchemaRuleCharsetAllowlist:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLCharsetAllowlist, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLEncodingAllowlist, nil
		}
	case SchemaRuleCollationAllowlist:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLCollationAllowlist, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLCollationAllowlist, nil
		}
	case SchemaRuleIndexPKTypeLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLIndexPKType, nil
		}
	case SchemaRuleIndexTypeNoBlob:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLIndexTypeNoBlob, nil
		}
	case SchemaRuleIndexPrimaryKeyTypeAllowlist:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_OCEANBASE:
			return MySQLPrimaryKeyTypeAllowlist, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLPrimaryKeyTypeAllowlist, nil
		}
	case SchemaRuleCreateIndexConcurrently:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLCreateIndexConcurrently, nil
		}
	case SchemaRuleIndexTypeAllowList:
		if engine == storepb.Engine_MYSQL {
			return MySQLIndexTypeAllowList, nil
		}
	case SchemaRuleStatementInsertRowLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLInsertRowLimit, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLInsertRowLimit, nil
		}
	case SchemaRuleStatementInsertMustSpecifyColumn:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLInsertMustSpecifyColumn, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLInsertMustSpecifyColumn, nil
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
			return OracleInsertMustSpecifyColumn, nil
		}
	case SchemaRuleStatementInsertDisallowOrderByRand:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLInsertDisallowOrderByRand, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLInsertDisallowOrderByRand, nil
		}
	case SchemaRuleStatementDisallowLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLStatementDisallowLimit, nil
		}
	case SchemaRuleStatementDisallowOrderBy:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLDisallowOrderBy, nil
		}
	case SchemaRuleStatementMergeAlterTable:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLMergeAlterTable, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLMergeAlterTable, nil
		}
	case SchemaRuleStatementAffectedRowLimit:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLStatementAffectedRowLimit, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLStatementAffectedRowLimit, nil
		}
	case SchemaRuleStatementDMLDryRun:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLStatementDMLDryRun, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLStatementDMLDryRun, nil
		case storepb.Engine_ORACLE:
			return OracleStatementDMLDryRun, nil
		}
	case SchemaRuleStatementDisallowAddColumnWithDefault:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLDisallowAddColumnWithDefault, nil
		}
	case SchemaRuleStatementAddCheckNotValid:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLAddCheckNotValid, nil
		}
	case SchemaRuleStatementAddFKNotValid:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLAddFKNotValid, nil
		}
	case SchemaRuleStatementDisallowAddNotNull:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLDisallowAddNotNull, nil
		}
	case SchemaRuleStatementSelectFullTableScan:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			return MySQLStatementSelectFullTableScan, nil
		}
	case SchemaRuleStatementCreateSpecifySchema:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLStatementCreateSpecifySchema, nil
		}
	case SchemaRuleStatementCheckSetRoleVariable:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLStatementCheckSetRoleVariable, nil
		}
	case SchemaRuleStatementWhereNoEqualNull:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementWhereNoEqualNull, nil
		}
	case SchemaRuleStatementWhereDisallowFunctionsAndCaculations:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementWhereDisallowUsingFunction, nil
		}
		if engine == storepb.Engine_MSSQL {
			return MSSQLStatementWhereDisallowFunctionsAndCalculations, nil
		}
		if engine == storepb.Engine_MSSQL {
			return MSSQLStatementWhereDisallowFunctionsAndCalculations, nil
		}
	case SchemaRuleStatementQueryMinumumPlanLevel:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementQueryMinumumPlanLevel, nil
		}
	case SchemaRuleStatementWhereMaximumLogicalOperatorCount:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementWhereMaximumLogicalOperatorCount, nil
		}
	case SchemaRuleStatementMaximumLimitValue:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE, storepb.Engine_TIDB:
			return MySQLStatementMaximumLimitValue, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLStatementMaximumLimitValue, nil
		}
	case SchemaRuleStatementMaximumJoinTableCount:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementMaximumJoinTableCount, nil
		}
	case SchemaRuleStatementMaximumStatementsInTransaction:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementMaximumStatementsInTransaction, nil
		}
	case SchemaRuleStatementJoinStrictColumnAttrs:
		if engine == storepb.Engine_MYSQL {
			return MySQLStatementJoinStrictColumnAttrs, nil
		}
	case SchemaRuleStatementDisallowCrossDBQueries:
		if engine == storepb.Engine_MSSQL {
			return MSSQLStatementDisallowCrossDBQueries, nil
		}
	case SchemaRuleStatementMaxExecutionTime:
		if engine == storepb.Engine_MYSQL || engine == storepb.Engine_MARIADB {
			return MySQLStatementMaxExecutionTime, nil
		}
	case SchemaRuleStatementRequireAlgorithmOption:
		if engine == storepb.Engine_MYSQL || engine == storepb.Engine_MARIADB {
			return MySQLStatementRequireAlgorithmOption, nil
		}
	case SchemaRuleStatementRequireLockOption:
		if engine == storepb.Engine_MYSQL || engine == storepb.Engine_MARIADB {
			return MySQLStatementRequireLockOption, nil
		}
	case SchemaRuleStatementObjectOwnerCheck:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLStatementObjectOwnerCheck, nil
		}
	case SchemaRuleCommentLength:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLCommentConvention, nil
		}
	case SchemaRuleProcedureDisallowCreate:
		if engine == storepb.Engine_MYSQL {
			return MySQLProcedureDisallowCreate, nil
		}
		if engine == storepb.Engine_MSSQL {
			return MSSQLProcedureDisallowCreateOrAlter, nil
		}
	case SchemaRuleEventDisallowCreate:
		if engine == storepb.Engine_MYSQL {
			return MySQLEventDisallowCreate, nil
		}
	case SchemaRuleViewDisallowCreate:
		if engine == storepb.Engine_MYSQL {
			return MySQLViewDisallowCreate, nil
		}
	case SchemaRuleFunctionDisallowCreate:
		if engine == storepb.Engine_MYSQL {
			return MySQLFunctionDisallowCreate, nil
		}
		if engine == storepb.Engine_MSSQL {
			return MSSQLFunctionDisallowCreateOrAlter, nil
		}
	case SchemaRuleFunctionDisallowList:
		if engine == storepb.Engine_MYSQL {
			return MySQLFunctionDisallowedList, nil
		}
	case SchemaRuleOnlineMigration:
		if engine == storepb.Engine_MYSQL || engine == storepb.Engine_MARIADB {
			return MySQLOnlineMigration, nil
		}
	case SchemaRuleStatementNonTransactional:
		if engine == storepb.Engine_POSTGRES {
			return PostgreSQLNonTransactional, nil
		}
	case SchemaRuleStatementDisallowOfflineDDL:
		if engine == storepb.Engine_OCEANBASE {
			return MySQLDisallowOfflineDDL, nil
		}
	// ----------------- Builtin Rules -----------------------
	case BuiltinRulePriorBackupCheck:
		switch engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB:
			return MySQLBuiltinPriorBackupCheck, nil
		case storepb.Engine_POSTGRES:
			return PostgreSQLBuiltinPriorBackupCheck, nil
		case storepb.Engine_MSSQL:
			return MSSQLBuiltinPriorBackupCheck, nil
		case storepb.Engine_ORACLE:
			return OracleBuiltinPriorBackupCheck, nil
		}
	}
	return "", errors.Errorf("unknown SQL review rule type %v for %v", ruleType, engine)
}
