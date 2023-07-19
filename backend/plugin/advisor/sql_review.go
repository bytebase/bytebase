package advisor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"

	// register pg parser.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

// How to add a SQL review rule:
//   1. Implement an advisor.(plugin/advisor/mysql or plugin/advisor/pg)
//   2. Register this advisor in map[db.Type][AdvisorType].(plugin/advisor.go)
//   3. Add advisor error code if needed(plugin/advisor/code.go).
//   4. Map SQLReviewRuleType to advisor.Type in getAdvisorTypeByRule(current file).

// SQLReviewRuleLevel is the error level for SQL review rule.
type SQLReviewRuleLevel string

// SQLReviewRuleType is the type of schema rule.
type SQLReviewRuleType string

const (
	// SchemaRuleLevelError is the error level of SQLReviewRuleLevel.
	SchemaRuleLevelError SQLReviewRuleLevel = "ERROR"
	// SchemaRuleLevelWarning is the warning level of SQLReviewRuleLevel.
	SchemaRuleLevelWarning SQLReviewRuleLevel = "WARNING"
	// SchemaRuleLevelDisabled is the disabled level of SQLReviewRuleLevel.
	SchemaRuleLevelDisabled SQLReviewRuleLevel = "DISABLED"

	// SchemaRuleMySQLEngine require InnoDB as the storage engine.
	SchemaRuleMySQLEngine SQLReviewRuleType = "engine.mysql.use-innodb"

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
	// SchemaRuleStatementRequireWhere require 'WHERE' clause.
	SchemaRuleStatementRequireWhere SQLReviewRuleType = "statement.where.require"
	// SchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	SchemaRuleStatementNoLeadingWildcardLike SQLReviewRuleType = "statement.where.no-leading-wildcard-like"
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
	// SchemaRuleStatementDisallowAddNotNull disallow to add NOT NULL.
	SchemaRuleStatementDisallowAddNotNull = "statement.disallow-add-not-null"

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
	// SchemaRuleAddNotNullColumnRequireDefault enforce the adding not null column requires default.
	SchemaRuleAddNotNullColumnRequireDefault SQLReviewRuleType = "column.add-not-null-require-default"

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

	// SchemaRuleCharsetAllowlist enforce the charset allowlist.
	SchemaRuleCharsetAllowlist SQLReviewRuleType = "system.charset.allowlist"

	// SchemaRuleCollationAllowlist enforce the collation allowlist.
	SchemaRuleCollationAllowlist SQLReviewRuleType = "system.collation.allowlist"

	// SchemaRuleCommentLength limit comment length.
	SchemaRuleCommentLength SQLReviewRuleType = "system.comment.length"

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

// SQLReviewPolicy is the policy configuration for SQL review.
type SQLReviewPolicy struct {
	Name     string           `json:"name"`
	RuleList []*SQLReviewRule `json:"ruleList"`
}

// Validate validates the SQLReviewPolicy. It also validates the each review rule.
func (policy *SQLReviewPolicy) Validate() error {
	if policy.Name == "" || len(policy.RuleList) == 0 {
		return errors.Errorf("invalid payload, name or rule list cannot be empty")
	}
	for _, rule := range policy.RuleList {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// String returns the marshal string value for SQL review policy.
func (policy *SQLReviewPolicy) String() (string, error) {
	s, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// SQLReviewRule is the rule for SQL review policy.
type SQLReviewRule struct {
	Type    SQLReviewRuleType  `json:"type"`
	Level   SQLReviewRuleLevel `json:"level"`
	Engine  db.Type            `json:"engine"`
	Comment string             `json:"comment"`
	// Payload is the stringify value for XXXRulePayload (e.g. NamingRulePayload, StringArrayTypeRulePayload)
	// If the rule doesn't have any payload configuration, the payload would be "{}"
	Payload string `json:"payload"`
}

// Validate validates the SQL review rule.
func (rule *SQLReviewRule) Validate() error {
	// TODO(rebelice): add other SQL review rule validation.
	switch rule.Type {
	case SchemaRuleTableNaming, SchemaRuleColumnNaming, SchemaRuleAutoIncrementColumnNaming:
		if _, _, err := UnmarshalNamingRulePayloadAsRegexp(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleFKNaming, SchemaRuleIDXNaming, SchemaRuleUKNaming:
		if _, _, _, err := UnmarshalNamingRulePayloadAsTemplate(rule.Type, rule.Payload); err != nil {
			return err
		}
	case SchemaRuleRequiredColumn:
		if _, err := UnmarshalRequiredColumnList(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleColumnCommentConvention, SchemaRuleTableCommentConvention:
		if _, err := UnmarshalCommentConventionRulePayload(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleIndexKeyNumberLimit, SchemaRuleStatementInsertRowLimit, SchemaRuleIndexTotalNumberLimit,
		SchemaRuleColumnMaximumCharacterLength, SchemaRuleColumnMaximumVarcharLength, SchemaRuleColumnAutoIncrementInitialValue, SchemaRuleStatementAffectedRowLimit:
		if _, err := UnmarshalNumberTypeRulePayload(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleColumnTypeDisallowList, SchemaRuleCharsetAllowlist, SchemaRuleCollationAllowlist, SchemaRuleIndexPrimaryKeyTypeAllowlist:
		if _, err := UnmarshalStringArrayTypeRulePayload(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleIdentifierCase:
		if _, err := UnmarshalNamingCaseRulePayload(rule.Payload); err != nil {
			return err
		}
	}
	return nil
}

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
	Required  bool `json:"required"`
	MaxLength int  `json:"maxLength"`
}

// NumberTypeRulePayload is the number type payload.
type NumberTypeRulePayload struct {
	Number int `json:"number"`
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
	Charset   string
	Collation string
	DbType    db.Type
	Catalog   catalog.Catalog
	Driver    *sql.DB
	Context   context.Context

	// Snowflake specific fields
	CurrentDatabase string
	// Oracle specific fields
	CurrentSchema string
}

func syntaxCheck(statement string, checkContext SQLReviewCheckContext) (any, []Advice) {
	switch checkContext.DbType {
	case db.MySQL, db.MariaDB, db.OceanBase, db.TiDB:
		return mysqlSyntaxCheck(statement)
	case db.Postgres:
		return postgresSyntaxCheck(statement)
	case db.Oracle:
		return oracleSyntaxCheck(statement)
	case db.Snowflake:
		return snowflakeSyntaxCheck(statement)
	case db.MSSQL:
		return mssqlSyntaxCheck(statement)
	}
	return nil, []Advice{
		{
			Status:  Error,
			Code:    Unsupported,
			Title:   "Unsupported database type",
			Content: fmt.Sprintf("Unsupported database type %s", checkContext.DbType),
			Line:    1,
		},
	}
}

func mssqlSyntaxCheck(statement string) (any, []Advice) {
	tree, err := parser.ParseTSQL(statement)
	if err != nil {
		if syntaxErr, ok := err.(*parser.SyntaxError); ok {
			return nil, []Advice{
				{
					Status:  Warn,
					Code:    StatementSyntaxError,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					Line:    syntaxErr.Line,
				},
			}
		}
		return nil, []Advice{
			{
				Status:  Warn,
				Code:    Internal,
				Title:   "Parse error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	return tree, nil
}

func snowflakeSyntaxCheck(statement string) (any, []Advice) {
	tree, err := parser.ParseSnowSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*parser.SyntaxError); ok {
			return nil, []Advice{
				{
					Status:  Warn,
					Code:    StatementSyntaxError,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					Line:    syntaxErr.Line,
				},
			}
		}
		return nil, []Advice{
			{
				Status:  Warn,
				Code:    Internal,
				Title:   "Parse error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	return tree, nil
}

func oracleSyntaxCheck(statement string) (any, []Advice) {
	tree, _, err := parser.ParsePLSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*parser.SyntaxError); ok {
			return nil, []Advice{
				{
					Status:  Warn,
					Code:    StatementSyntaxError,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					Line:    syntaxErr.Line,
				},
			}
		}
		return nil, []Advice{
			{
				Status:  Warn,
				Code:    Internal,
				Title:   "Parse error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	return tree, nil
}

func postgresSyntaxCheck(statement string) (any, []Advice) {
	nodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		if _, ok := err.(*parser.ConvertError); ok {
			return nil, []Advice{
				{
					Status:  Error,
					Code:    Internal,
					Title:   "Parser conversion error",
					Content: err.Error(),
					Line:    calculatePostgresErrorLine(statement),
				},
			}
		}
		return nil, []Advice{
			{
				Status:  Error,
				Code:    StatementSyntaxError,
				Title:   SyntaxErrorTitle,
				Content: err.Error(),
				Line:    calculatePostgresErrorLine(statement),
			},
		}
	}
	var res []ast.Node
	for _, node := range nodes {
		if node != nil {
			res = append(res, node)
		}
	}
	return res, nil
}

func calculatePostgresErrorLine(statement string) int {
	statementList, err := parser.SplitMultiSQL(parser.Postgres, statement)
	if err != nil {
		// nolint:nilerr
		return 1
	}

	for _, stmt := range statementList {
		if _, err := parser.Parse(parser.Postgres, parser.ParseContext{}, stmt.Text); err != nil {
			return stmt.LastLine
		}
	}

	return 0
}

func newTiDBParser() *tidbparser.Parser {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

func mysqlSyntaxCheck(statement string) (any, []Advice) {
	list, err := parser.SplitMySQL(statement)
	if err != nil {
		return nil, []Advice{
			{
				Status:  Warn,
				Code:    Internal,
				Title:   "Syntax error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	p := newTiDBParser()
	var returnNodes []tidbast.StmtNode
	var adviceList []Advice
	for _, item := range list {
		nodes, _, err := p.Parse(item.Text, "", "")
		if err != nil {
			// TiDB parser doesn't fully support MySQL syntax, so we need to use MySQL parser to parse the statement.
			// But MySQL parser has some performance issue, so we only use it to parse the statement after TiDB parser failed.
			if _, err := parser.ParseMySQL(item.Text); err != nil {
				if syntaxErr, ok := err.(*parser.SyntaxError); ok {
					return nil, []Advice{
						{
							Status:  Error,
							Code:    StatementSyntaxError,
							Title:   SyntaxErrorTitle,
							Content: syntaxErr.Message,
							Line:    syntaxErr.Line,
						},
					}
				}
				return nil, []Advice{
					{
						Status:  Warn,
						Code:    Internal,
						Title:   "Parse error",
						Content: err.Error(),
						Line:    1,
					},
				}
			}
			// If MySQL parser can parse the statement, but TiDB parser can't, we just ignore the statement.
			continue
		}

		if len(nodes) != 1 {
			continue
		}

		node := nodes[0]
		node.SetText(nil, item.Text)
		node.SetOriginTextPosition(item.LastLine)
		if n, ok := node.(*tidbast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, append(adviceList, Advice{
					Status:  Error,
					Code:    Internal,
					Title:   "Set line error",
					Content: err.Error(),
					Line:    item.LastLine,
				})
			}
		}
		returnNodes = append(returnNodes, node)
	}

	return returnNodes, adviceList
}

// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(statements string, ruleList []*SQLReviewRule, checkContext SQLReviewCheckContext) ([]Advice, error) {
	ast, result := syntaxCheck(statements, checkContext)
	if ast == nil || len(ruleList) == 0 {
		return result, nil
	}

	finder := checkContext.Catalog.GetFinder()
	switch checkContext.DbType {
	case db.TiDB, db.MySQL, db.MariaDB, db.Postgres, db.OceanBase:
		if err := finder.WalkThrough(statements); err != nil {
			return convertWalkThroughErrorToAdvice(checkContext, err)
		}
	}

	for _, rule := range ruleList {
		if rule.Engine != "" && rule.Engine != checkContext.DbType {
			continue
		}
		if rule.Level == SchemaRuleLevelDisabled {
			continue
		}

		advisorType, err := getAdvisorTypeByRule(rule.Type, checkContext.DbType)
		if err != nil {
			if rule.Engine != "" {
				log.Warn("not supported rule", zap.String("rule type", string(rule.Type)), zap.String("engine", string(rule.Engine)), zap.Error(err))
			}
			continue
		}

		adviceList, err := Check(
			checkContext.DbType,
			advisorType,
			Context{
				Charset:         checkContext.Charset,
				Collation:       checkContext.Collation,
				AST:             ast,
				Rule:            rule,
				Catalog:         finder,
				Driver:          checkContext.Driver,
				Context:         checkContext.Context,
				CurrentSchema:   checkContext.CurrentSchema,
				CurrentDatabase: checkContext.CurrentDatabase,
			},
			statements,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check statement")
		}

		result = append(result, adviceList...)
	}

	// There may be multiple syntax errors, return one only.
	if len(result) > 0 && result[0].Title == SyntaxErrorTitle {
		return result[:1], nil
	}
	sort.SliceStable(result, func(i, j int) bool {
		// Error is 2, warning is 1. So the error (value 2) should come first.
		return result[i].Status.GetPriority() > result[j].Status.GetPriority()
	})
	if len(result) == 0 {
		result = append(result, Advice{
			Status:  Success,
			Code:    Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return result, nil
}

func convertWalkThroughErrorToAdvice(checkContext SQLReviewCheckContext, err error) ([]Advice, error) {
	walkThroughError, ok := err.(*catalog.WalkThroughError)
	if !ok {
		return nil, err
	}

	var res []Advice
	switch walkThroughError.Type {
	case catalog.ErrorTypeUnsupported:
		res = append(res, Advice{
			Status:  Error,
			Code:    Unsupported,
			Title:   walkThroughError.Content,
			Content: "",
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeParseError:
		res = append(res, Advice{
			Status:  Error,
			Code:    StatementSyntaxError,
			Title:   SyntaxErrorTitle,
			Content: walkThroughError.Content,
		})
	case catalog.ErrorTypeDeparseError:
		res = append(res, Advice{
			Status:  Error,
			Code:    Internal,
			Title:   "Internal error for walk-through",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeAccessOtherDatabase:
		res = append(res, Advice{
			Status:  Error,
			Code:    NotCurrentDatabase,
			Title:   "Access other database",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeDatabaseIsDeleted:
		res = append(res, Advice{
			Status:  Error,
			Code:    DatabaseIsDeleted,
			Title:   "Access deleted database",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeTableExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    TableExists,
			Title:   "Table already exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeTableNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    TableNotExists,
			Title:   "Table does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeColumnExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    ColumnExists,
			Title:   "Column already exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeColumnNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    ColumnNotExists,
			Title:   "Column does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeDropAllColumns:
		res = append(res, Advice{
			Status:  Error,
			Code:    DropAllColumns,
			Title:   "Drop all columns",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypePrimaryKeyExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    PrimaryKeyExists,
			Title:   "Primary key exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIndexExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    IndexExists,
			Title:   "Index exists",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIndexEmptyKeys:
		res = append(res, Advice{
			Status:  Error,
			Code:    IndexEmptyKeys,
			Title:   "Index empty keys",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypePrimaryKeyNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    PrimaryKeyNotExists,
			Title:   "Primary key does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIndexNotExists:
		res = append(res, Advice{
			Status:  Error,
			Code:    IndexNotExists,
			Title:   "Index does not exist",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeIncorrectIndexName:
		res = append(res, Advice{
			Status:  Error,
			Code:    IncorrectIndexName,
			Title:   "Incorrect index name",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeSpatialIndexKeyNullable:
		res = append(res, Advice{
			Status:  Error,
			Code:    SpatialIndexKeyNullable,
			Title:   "Spatial index key must be NOT NULL",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
		})
	case catalog.ErrorTypeColumnIsReferencedByView:
		details := ""
		if checkContext.DbType == db.Postgres {
			list, yes := walkThroughError.Payload.([]string)
			if !yes {
				return nil, errors.Errorf("invalid payload for ColumnIsReferencedByView, expect []string but found %T", walkThroughError.Payload)
			}
			if definition, err := getViewDefinition(checkContext, list); err != nil {
				log.Warn("failed to get view definition", zap.Error(err))
			} else {
				details = definition
			}
		}

		res = append(res, Advice{
			Status:  Error,
			Code:    ColumnIsReferencedByView,
			Title:   "Column is referenced by view",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
			Details: details,
		})
	case catalog.ErrorTypeTableIsReferencedByView:
		details := ""
		if checkContext.DbType == db.Postgres {
			list, yes := walkThroughError.Payload.([]string)
			if !yes {
				return nil, errors.Errorf("invalid payload for TableIsReferencedByView, expect []string but found %T", walkThroughError.Payload)
			}
			if definition, err := getViewDefinition(checkContext, list); err != nil {
				log.Warn("failed to get view definition", zap.Error(err))
			} else {
				details = definition
			}
		}

		res = append(res, Advice{
			Status:  Error,
			Code:    TableIsReferencedByView,
			Title:   "Table is referenced by view",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
			Details: details,
		})
	default:
		res = append(res, Advice{
			Status:  Error,
			Code:    Internal,
			Title:   "Failed to walk-through",
			Content: walkThroughError.Content,
			Line:    walkThroughError.Line,
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

// RuleExists returns true if rule exists.
func RuleExists(ruleType SQLReviewRuleType, engine db.Type) bool {
	_, err := getAdvisorTypeByRule(ruleType, engine)
	return err == nil
}

func getAdvisorTypeByRule(ruleType SQLReviewRuleType, engine db.Type) (Type, error) {
	switch ruleType {
	case SchemaRuleStatementRequireWhere:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLWhereRequirement, nil
		case db.Postgres:
			return PostgreSQLWhereRequirement, nil
		case db.Oracle:
			return OracleWhereRequirement, nil
		case db.Snowflake:
			return SnowflakeWhereRequirement, nil
		}
	case SchemaRuleStatementNoLeadingWildcardLike:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNoLeadingWildcardLike, nil
		case db.Postgres:
			return PostgreSQLNoLeadingWildcardLike, nil
		case db.Oracle:
			return OracleNoLeadingWildcardLike, nil
		}
	case SchemaRuleStatementNoSelectAll:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNoSelectAll, nil
		case db.Postgres:
			return PostgreSQLNoSelectAll, nil
		case db.Oracle:
			return OracleNoSelectAll, nil
		case db.Snowflake:
			return SnowflakeNoSelectAll, nil
		case db.MSSQL:
			return MSSQLNoSelectAll, nil
		}
	case SchemaRuleSchemaBackwardCompatibility:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLMigrationCompatibility, nil
		case db.Postgres:
			return PostgreSQLMigrationCompatibility, nil
		case db.Snowflake:
			return SnowflakeMigrationCompatibility, nil
		}
	case SchemaRuleTableNaming:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNamingTableConvention, nil
		case db.Postgres:
			return PostgreSQLNamingTableConvention, nil
		case db.Oracle:
			return OracleNamingTableConvention, nil
		case db.Snowflake:
			return SnowflakeNamingTableConvention, nil
		case db.MSSQL:
			return MSSQLNamingTableConvention, nil
		}
	case SchemaRuleIDXNaming:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNamingIndexConvention, nil
		case db.Postgres:
			return PostgreSQLNamingIndexConvention, nil
		}
	case SchemaRulePKNaming:
		if engine == db.Postgres {
			return PostgreSQLNamingPKConvention, nil
		}
	case SchemaRuleUKNaming:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNamingUKConvention, nil
		case db.Postgres:
			return PostgreSQLNamingUKConvention, nil
		}
	case SchemaRuleFKNaming:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNamingFKConvention, nil
		case db.Postgres:
			return PostgreSQLNamingFKConvention, nil
		}
	case SchemaRuleColumnNaming:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNamingColumnConvention, nil
		case db.Postgres:
			return PostgreSQLNamingColumnConvention, nil
		}
	case SchemaRuleAutoIncrementColumnNaming:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLNamingAutoIncrementColumnConvention, nil
		}
	case SchemaRuleTableNameNoKeyword:
		switch engine {
		case db.Oracle:
			return OracleTableNamingNoKeyword, nil
		case db.Snowflake:
			return SnowflakeTableNamingNoKeyword, nil
		case db.MSSQL:
			return MSSQLTableNamingNoKeyword, nil
		}
	case SchemaRuleIdentifierNoKeyword:
		switch engine {
		case db.Oracle:
			return OracleIdentifierNamingNoKeyword, nil
		case db.Snowflake:
			return SnowflakeIdentifierNamingNoKeyword, nil
		}
	case SchemaRuleIdentifierCase:
		switch engine {
		case db.Oracle:
			return OracleIdentifierCase, nil
		case db.Snowflake:
			return SnowflakeIdentifierCase, nil
		}
	case SchemaRuleRequiredColumn:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnRequirement, nil
		case db.Postgres:
			return PostgreSQLColumnRequirement, nil
		case db.Oracle:
			return OracleColumnRequirement, nil
		case db.Snowflake:
			return SnowflakeColumnRequirement, nil
		}
	case SchemaRuleColumnNotNull:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnNoNull, nil
		case db.Postgres:
			return PostgreSQLColumnNoNull, nil
		case db.Oracle:
			return OracleColumnNoNull, nil
		case db.Snowflake:
			return SnowflakeColumnNoNull, nil
		}
	case SchemaRuleColumnDisallowChangeType:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnDisallowChangingType, nil
		case db.Postgres:
			return PostgreSQLColumnDisallowChangingType, nil
		}
	case SchemaRuleColumnSetDefaultForNotNull:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnSetDefaultForNotNull, nil
		}
	case SchemaRuleColumnDisallowChange:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnDisallowChanging, nil
		}
	case SchemaRuleColumnDisallowChangingOrder:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnDisallowChangingOrder, nil
		}
	case SchemaRuleColumnCommentConvention:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnCommentConvention, nil
		}
	case SchemaRuleColumnAutoIncrementMustInteger:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLAutoIncrementColumnMustInteger, nil
		}
	case SchemaRuleColumnTypeDisallowList:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnTypeRestriction, nil
		case db.Postgres:
			return PostgreSQLColumnTypeDisallowList, nil
		case db.Oracle:
			return OracleColumnTypeDisallowList, nil
		}
	case SchemaRuleColumnDisallowSetCharset:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLDisallowSetColumnCharset, nil
		}
	case SchemaRuleColumnMaximumCharacterLength:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLColumnMaximumCharacterLength, nil
		case db.Postgres:
			return PostgreSQLColumnMaximumCharacterLength, nil
		case db.Oracle:
			return OracleColumnMaximumCharacterLength, nil
		}
	case SchemaRuleColumnMaximumVarcharLength:
		switch engine {
		case db.Oracle:
			return OracleColumnMaximumVarcharLength, nil
		case db.Snowflake:
			return SnowflakeColumnMaximumVarcharLength, nil
		}
	case SchemaRuleColumnAutoIncrementInitialValue:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLAutoIncrementColumnInitialValue, nil
		}
	case SchemaRuleColumnAutoIncrementMustUnsigned:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLAutoIncrementColumnMustUnsigned, nil
		}
	case SchemaRuleCurrentTimeColumnCountLimit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLCurrentTimeColumnCountLimit, nil
		}
	case SchemaRuleColumnRequireDefault:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLRequireColumnDefault, nil
		case db.Postgres:
			return PostgreSQLRequireColumnDefault, nil
		case db.Oracle:
			return OracleRequireColumnDefault, nil
		}
	case SchemaRuleAddNotNullColumnRequireDefault:
		if engine == db.Oracle {
			return OracleAddNotNullColumnRequireDefault, nil
		}
	case SchemaRuleTableRequirePK:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLTableRequirePK, nil
		case db.Postgres:
			return PostgreSQLTableRequirePK, nil
		case db.Oracle:
			return OracleTableRequirePK, nil
		case db.Snowflake:
			return SnowflakeTableRequirePK, nil
		}
	case SchemaRuleTableNoFK:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLTableNoFK, nil
		case db.Postgres:
			return PostgreSQLTableNoFK, nil
		case db.Oracle:
			return OracleTableNoFK, nil
		case db.Snowflake:
			return SnowflakeTableNoFK, nil
		}
	case SchemaRuleTableDropNamingConvention:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLTableDropNamingConvention, nil
		case db.Postgres:
			return PostgreSQLTableDropNamingConvention, nil
		case db.Snowflake:
			return SnowflakeTableDropNamingConvention, nil
		}
	case SchemaRuleTableCommentConvention:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLTableCommentConvention, nil
		}
	case SchemaRuleTableDisallowPartition:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLTableDisallowPartition, nil
		case db.Postgres:
			return PostgreSQLTableDisallowPartition, nil
		}
	case SchemaRuleMySQLEngine:
		switch engine {
		case db.MySQL, db.MariaDB:
			return MySQLUseInnoDB, nil
		}
	case SchemaRuleDropEmptyDatabase:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLDatabaseAllowDropIfEmpty, nil
		}
	case SchemaRuleIndexNoDuplicateColumn:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLIndexNoDuplicateColumn, nil
		case db.Postgres:
			return PostgreSQLIndexNoDuplicateColumn, nil
		}
	case SchemaRuleIndexKeyNumberLimit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLIndexKeyNumberLimit, nil
		case db.Postgres:
			return PostgreSQLIndexKeyNumberLimit, nil
		case db.Oracle:
			return OracleIndexKeyNumberLimit, nil
		}
	case SchemaRuleIndexTotalNumberLimit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLIndexTotalNumberLimit, nil
		case db.Postgres:
			return PostgreSQLIndexTotalNumberLimit, nil
		}
	case SchemaRuleStatementDisallowCommit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLStatementDisallowCommit, nil
		case db.Postgres:
			return PostgreSQLStatementDisallowCommit, nil
		}
	case SchemaRuleCharsetAllowlist:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLCharsetAllowlist, nil
		case db.Postgres:
			return PostgreSQLEncodingAllowlist, nil
		}
	case SchemaRuleCollationAllowlist:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLCollationAllowlist, nil
		case db.Postgres:
			return PostgreSQLCollationAllowlist, nil
		}
	case SchemaRuleIndexPKTypeLimit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLIndexPKType, nil
		}
	case SchemaRuleIndexTypeNoBlob:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLIndexTypeNoBlob, nil
		}
	case SchemaRuleIndexPrimaryKeyTypeAllowlist:
		switch engine {
		case db.MySQL, db.TiDB, db.OceanBase:
			return MySQLPrimaryKeyTypeAllowlist, nil
		case db.Postgres:
			return PostgreSQLPrimaryKeyTypeAllowlist, nil
		}
	case SchemaRuleCreateIndexConcurrently:
		if engine == db.Postgres {
			return PostgreSQLCreateIndexConcurrently, nil
		}
	case SchemaRuleStatementInsertRowLimit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLInsertRowLimit, nil
		case db.Postgres:
			return PostgreSQLInsertRowLimit, nil
		}
	case SchemaRuleStatementInsertMustSpecifyColumn:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLInsertMustSpecifyColumn, nil
		case db.Postgres:
			return PostgreSQLInsertMustSpecifyColumn, nil
		case db.Oracle:
			return OracleInsertMustSpecifyColumn, nil
		}
	case SchemaRuleStatementInsertDisallowOrderByRand:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLInsertDisallowOrderByRand, nil
		case db.Postgres:
			return PostgreSQLInsertDisallowOrderByRand, nil
		}
	case SchemaRuleStatementDisallowLimit:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLDisallowLimit, nil
		}
	case SchemaRuleStatementDisallowOrderBy:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLDisallowOrderBy, nil
		}
	case SchemaRuleStatementMergeAlterTable:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLMergeAlterTable, nil
		case db.Postgres:
			return PostgreSQLMergeAlterTable, nil
		}
	case SchemaRuleStatementAffectedRowLimit:
		switch engine {
		case db.MySQL, db.MariaDB, db.OceanBase:
			return MySQLStatementAffectedRowLimit, nil
		case db.Postgres:
			return PostgreSQLStatementAffectedRowLimit, nil
		}
	case SchemaRuleStatementDMLDryRun:
		switch engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			return MySQLStatementDMLDryRun, nil
		case db.Postgres:
			return PostgreSQLStatementDMLDryRun, nil
		}
	case SchemaRuleStatementDisallowAddColumnWithDefault:
		if engine == db.Postgres {
			return PostgreSQLDisallowAddColumnWithDefault, nil
		}
	case SchemaRuleStatementAddCheckNotValid:
		if engine == db.Postgres {
			return PostgreSQLAddCheckNotValid, nil
		}
	case SchemaRuleStatementDisallowAddNotNull:
		if engine == db.Postgres {
			return PostgreSQLDisallowAddNotNull, nil
		}
	case SchemaRuleCommentLength:
		if engine == db.Postgres {
			return PostgreSQLCommentConvention, nil
		}
	}
	return Fake, errors.Errorf("unknown SQL review rule type %v for %v", ruleType, engine)
}
