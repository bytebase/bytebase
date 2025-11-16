package advisor

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// How to add a SQL review rule:
//   1. Implement an advisor.(plugin/advisor/mysql or plugin/advisor/pg)
//   2. Register this advisor in map[storepb.Engine][SQLReviewRuleType].(plugin/advisor.go)
//   3. Add advisor error code if needed(plugin/advisor/code.go).

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
	SchemaRuleStatementDisallowAddColumnWithDefault SQLReviewRuleType = "statement.disallow-add-column-with-default"
	// SchemaRuleStatementAddCheckNotValid require add check constraints not valid.
	SchemaRuleStatementAddCheckNotValid SQLReviewRuleType = "statement.add-check-not-valid"
	// SchemaRuleStatementAddFKNotValid require add foreign key not valid.
	SchemaRuleStatementAddFKNotValid SQLReviewRuleType = "statement.add-foreign-key-not-valid"
	// SchemaRuleStatementDisallowAddNotNull disallow to add NOT NULL.
	SchemaRuleStatementDisallowAddNotNull SQLReviewRuleType = "statement.disallow-add-not-null"
	// SchemaRuleStatementSelectFullTableScan disallow full table scan.
	SchemaRuleStatementSelectFullTableScan SQLReviewRuleType = "statement.select-full-table-scan"
	// SchemaRuleStatementCreateSpecifySchema disallow to create table without specifying schema.
	SchemaRuleStatementCreateSpecifySchema SQLReviewRuleType = "statement.create-specify-schema"
	// SchemaRuleStatementCheckSetRoleVariable require add a check for SET ROLE variable.
	SchemaRuleStatementCheckSetRoleVariable SQLReviewRuleType = "statement.check-set-role-variable"
	// SchemaRuleStatementDisallowUsingFilesort disallow using filesort in execution plan.
	SchemaRuleStatementDisallowUsingFilesort SQLReviewRuleType = "statement.disallow-using-filesort"
	// SchemaRuleStatementDisallowUsingTemporary disallow using temporary in execution plan.
	SchemaRuleStatementDisallowUsingTemporary SQLReviewRuleType = "statement.disallow-using-temporary"
	// SchemaRuleStatementWhereNoEqualNull check the WHERE clause no equal null.
	SchemaRuleStatementWhereNoEqualNull SQLReviewRuleType = "statement.where.no-equal-null"
	// SchemaRuleStatementWhereDisallowFunctionsAndCalculations disallow using function in WHERE clause.
	SchemaRuleStatementWhereDisallowFunctionsAndCalculations SQLReviewRuleType = "statement.where.disallow-functions-and-calculations"
	// SchemaRuleStatementQueryMinumumPlanLevel enforce the minimum plan level.
	SchemaRuleStatementQueryMinumumPlanLevel SQLReviewRuleType = "statement.query.minimum-plan-level"
	// SchemaRuleStatementWhereMaximumLogicalOperatorCount enforce the maximum logical operator count in WHERE clause.
	SchemaRuleStatementWhereMaximumLogicalOperatorCount SQLReviewRuleType = "statement.where.maximum-logical-operator-count"
	// SchemaRuleStatementMaximumLimitValue enforce the maximum limit value.
	SchemaRuleStatementMaximumLimitValue SQLReviewRuleType = "statement.maximum-limit-value"
	// SchemaRuleStatementMaximumJoinTableCount enforce the maximum join table count in the statement.
	SchemaRuleStatementMaximumJoinTableCount SQLReviewRuleType = "statement.maximum-join-table-count"
	// SchemaRuleStatementMaximumStatementsInTransaction enforce the maximum statements in transaction.
	SchemaRuleStatementMaximumStatementsInTransaction SQLReviewRuleType = "statement.maximum-statements-in-transaction"
	// SchemaRuleStatementJoinStrictColumnAttrs enforce the join strict column attributes.
	SchemaRuleStatementJoinStrictColumnAttrs SQLReviewRuleType = "statement.join-strict-column-attrs"
	// SchemaRuleStatementDisallowMixInDDL disallows DML statements in DDL statements.
	SchemaRuleStatementDisallowMixInDDL SQLReviewRuleType = "statement.disallow-mix-in-ddl"
	// SchemaRuleStatementDisallowMixInDML disallows DDL statements in DML statements.
	SchemaRuleStatementDisallowMixInDML SQLReviewRuleType = "statement.disallow-mix-in-dml"
	// SchemaRuleStatementPriorBackupCheck checks for prior backup.
	SchemaRuleStatementPriorBackupCheck SQLReviewRuleType = "statement.prior-backup-check"
	// SchemaRuleStatementNonTransactional checks for non-transactional statements.
	SchemaRuleStatementNonTransactional SQLReviewRuleType = "statement.non-transactional"
	// SchemaRuleStatementAddColumnWithoutPosition check no position in ADD COLUMN clause.
	SchemaRuleStatementAddColumnWithoutPosition SQLReviewRuleType = "statement.add-column-without-position"
	// SchemaRuleStatementDisallowOfflineDDL disallow offline ddl.
	SchemaRuleStatementDisallowOfflineDDL SQLReviewRuleType = "statement.disallow-offline-ddl"
	// SchemaRuleStatementDisallowCrossDBQueries disallow cross database queries.
	SchemaRuleStatementDisallowCrossDBQueries SQLReviewRuleType = "statement.disallow-cross-db-queries"
	// SchemaRuleStatementMaxExecutionTime enforce the maximum execution time.
	SchemaRuleStatementMaxExecutionTime SQLReviewRuleType = "statement.max-execution-time"
	// SchemaRuleStatementRequireAlgorithmOption require set ALGORITHM option in ALTER TABLE statement.
	SchemaRuleStatementRequireAlgorithmOption SQLReviewRuleType = "statement.require-algorithm-option"
	// SchemaRuleStatementRequireLockOption require set LOCK option in ALTER TABLE statement.
	SchemaRuleStatementRequireLockOption SQLReviewRuleType = "statement.require-lock-option"
	// SchemaRuleStatementObjectOwnerCheck checks the object owner for the statement.
	SchemaRuleStatementObjectOwnerCheck SQLReviewRuleType = "statement.object-owner-check"
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

// SQLReviewCheckContext is the context for SQL review check.
type SQLReviewCheckContext struct {
	Charset               string
	Collation             string
	ChangeType            storepb.PlanCheckRunConfig_ChangeDatabaseType
	DBSchema              *storepb.DatabaseSchemaMetadata
	DBType                storepb.Engine
	OriginCatalog         *catalog.DatabaseState
	FinalCatalog          *catalog.DatabaseState
	Driver                *sql.DB
	EnablePriorBackup     bool
	ClassificationConfig  *storepb.DataClassificationSetting_DataClassificationConfig
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string
	IsObjectCaseSensitive bool

	// Snowflake specific fields
	CurrentDatabase string

	// Used for test only.
	NoAppendBuiltin bool

	// UsePostgresDatabaseOwner is true if the advisor should use the database owner as default role.
	UsePostgresDatabaseOwner bool
}

// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(
	ctx context.Context,
	sm *sheet.Manager,
	statements string,
	ruleList []*storepb.SQLReviewRule,
	checkContext SQLReviewCheckContext,
) ([]*storepb.Advice, error) {
	asts, parseResult := sm.GetASTsForChecks(checkContext.DBType, statements)

	builtinOnly := len(ruleList) == 0

	if !checkContext.NoAppendBuiltin {
		// Append builtin rules to the rule list.
		ruleList = append(ruleList, GetBuiltinRules(checkContext.DBType)...)
	}

	if asts == nil || len(ruleList) == 0 {
		return parseResult, nil
	}

	if !builtinOnly && checkContext.FinalCatalog != nil {
		switch checkContext.DBType {
		case storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
			if err := catalog.WalkThrough(checkContext.FinalCatalog, asts); err != nil {
				return convertWalkThroughErrorToAdvice(err)
			}
		default:
			// Other database types don't need walkthrough
		}
	}

	var errorAdvices, warningAdvices []*storepb.Advice
	for _, rule := range ruleList {
		if rule.Engine != storepb.Engine_ENGINE_UNSPECIFIED && rule.Engine != checkContext.DBType {
			continue
		}

		ruleType := SQLReviewRuleType(rule.Type)

		adviceList, err := Check(
			ctx,
			checkContext.DBType,
			ruleType,
			Context{
				DBSchema:                 checkContext.DBSchema,
				ChangeType:               checkContext.ChangeType,
				EnablePriorBackup:        checkContext.EnablePriorBackup,
				AST:                      asts,
				Statements:               statements,
				Rule:                     rule,
				OriginCatalog:            checkContext.OriginCatalog,
				FinalCatalog:             checkContext.FinalCatalog,
				Driver:                   checkContext.Driver,
				CurrentDatabase:          checkContext.CurrentDatabase,
				ClassificationConfig:     checkContext.ClassificationConfig,
				UsePostgresDatabaseOwner: checkContext.UsePostgresDatabaseOwner,
				ListDatabaseNamesFunc:    checkContext.ListDatabaseNamesFunc,
				InstanceID:               checkContext.InstanceID,
				IsObjectCaseSensitive:    checkContext.IsObjectCaseSensitive,
			},
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

func convertWalkThroughErrorToAdvice(err error) ([]*storepb.Advice, error) {
	walkThroughError, ok := err.(*catalog.WalkThroughError)
	if !ok {
		return nil, err
	}

	// Determine the advice status based on the error code
	// Most errors are ERROR level, except for a few special cases
	status := storepb.Advice_ERROR
	if walkThroughError.Code == code.ReferenceOtherDatabase {
		status = storepb.Advice_WARNING
	}

	return []*storepb.Advice{
		{
			Status:        status,
			Code:          walkThroughError.Code.Int32(),
			Title:         walkThroughError.Content,
			Content:       walkThroughError.Content,
			StartPosition: common.ConvertANTLRLineToPosition(walkThroughError.Line),
		},
	}, nil
}
