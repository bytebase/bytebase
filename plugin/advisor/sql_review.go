package advisor

import (
	"encoding/json"
	"log"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/advisor/db"
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

	// SchemaRuleStatementNoSelectAll disallow 'SELECT *'.
	SchemaRuleStatementNoSelectAll SQLReviewRuleType = "statement.select.no-select-all"
	// SchemaRuleStatementRequireWhere require 'WHERE' clause.
	SchemaRuleStatementRequireWhere SQLReviewRuleType = "statement.where.require"
	// SchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	SchemaRuleStatementNoLeadingWildcardLike SQLReviewRuleType = "statement.where.no-leading-wildcard-like"
	// SchemaRuleStatementNoCreateTableAs disallow 'CREATE TABLE ... [AS] SELECT.
	SchemaRuleStatementNoCreateTableAs SQLReviewRuleType = "statement.create-table.no-create-table-as"

	// SchemaRuleTableRequirePK require the table to have a primary key.
	SchemaRuleTableRequirePK SQLReviewRuleType = "table.require-pk"
	// SchemaRuleTableNoFK require the table disallow the foreign key.
	SchemaRuleTableNoFK SQLReviewRuleType = "table.no-foreign-key"
	// SchemaRuleTableDropNamingConvention require only the table following the naming convention can be deleted.
	SchemaRuleTableDropNamingConvention SQLReviewRuleType = "table.drop-naming-convention"
	// SchemaRuleTableExists require the table existence.
	SchemaRuleTableExists SQLReviewRuleType = "table.exists"
	// SchemaRuleTableCommentConvention enforce the table comment convention.
	SchemaRuleTableCommentConvention SQLReviewRuleType = "table.comment"
	// SchemaRuleTableNotExists check the table name conflict.
	SchemaRuleTableNotExists SQLReviewRuleType = "table.not-exists"

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
	// SchemaRuleColumnExists require the column existence.
	SchemaRuleColumnExists SQLReviewRuleType = "column.exists"
	// SchemaRuleColumnDisallowChangingOrder disallow changing column order.
	SchemaRuleColumnDisallowChangingOrder SQLReviewRuleType = "column.disallow-changing-order"
	// SchemaRuleColumnCommentConvention enforce the column comment convention.
	SchemaRuleColumnCommentConvention SQLReviewRuleType = "column.comment"
	// SchemaRuleColumnAutoIncrementMustInteger require the auto-increment column to be integer.
	SchemaRuleColumnAutoIncrementMustInteger SQLReviewRuleType = "column.auto-increment-must-integer"
	// SchemaRuleColumnTypeRestriction enforce the column type restriction.
	SchemaRuleColumnTypeRestriction SQLReviewRuleType = "column.type-restriction"

	// SchemaRuleSchemaBackwardCompatibility enforce the MySQL and TiDB support check whether the schema change is backward compatible.
	SchemaRuleSchemaBackwardCompatibility SQLReviewRuleType = "schema.backward-compatibility"

	// SchemaRuleDropEmptyDatabase enforce the MySQL and TiDB support check if the database is empty before users drop it.
	SchemaRuleDropEmptyDatabase SQLReviewRuleType = "database.drop-empty-database"

	// SchemaRuleIndexNoDuplicateColumn require the index no duplicate column.
	SchemaRuleIndexNoDuplicateColumn SQLReviewRuleType = "index.no-duplicate-column"
	// SchemaRuleIndexKeyNumberLimit enforce the index key number limit.
	SchemaRuleIndexKeyNumberLimit SQLReviewRuleType = "index.key-number-limit"
	// SchemaRuleIndexPKType enforce the type restriction of columns in primary key.
	SchemaRuleIndexPKType SQLReviewRuleType = "index.pk-type"
	// SchemaRuleIndexTypeNoBlob enforce the type restriction of columns in index.
	SchemaRuleIndexTypeNoBlob SQLReviewRuleType = "index.type-no-blob"
	// SchemaRuleIndexNotExists check the index name conflict.
	SchemaRuleIndexNotExists SQLReviewRuleType = "index.not-exists"

	// SchemaRuleCharsetAllowlist enforce the charset allowlist.
	SchemaRuleCharsetAllowlist SQLReviewRuleType = "charset.allowlist"

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

// SQLReviewRule is the rule for SQL review policy.
type SQLReviewRule struct {
	Type  SQLReviewRuleType  `json:"type"`
	Level SQLReviewRuleLevel `json:"level"`
	// Payload is the stringify value for XXXRulePayload (e.g. NamingRulePayload, RequiredColumnRulePayload)
	// If the rule doesn't have any payload configuration, the payload would be "{}"
	Payload string `json:"payload"`
}

// Validate validates the SQL review rule.
func (rule *SQLReviewRule) Validate() error {
	// TODO(rebelice): add other SQL review rule validation.
	switch rule.Type {
	case SchemaRuleTableNaming, SchemaRuleColumnNaming:
		if _, _, err := UnamrshalNamingRulePayloadAsRegexp(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleFKNaming, SchemaRuleIDXNaming, SchemaRuleUKNaming:
		if _, _, _, err := UnmarshalNamingRulePayloadAsTemplate(rule.Type, rule.Payload); err != nil {
			return err
		}
	case SchemaRuleRequiredColumn:
		if _, err := UnmarshalRequiredColumnRulePayload(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleColumnCommentConvention, SchemaRuleTableCommentConvention:
		if _, err := UnmarshalCommentConventionRulePayload(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleIndexKeyNumberLimit:
		if _, err := UnmarshalNumberLimitRulePayload(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleColumnTypeRestriction:
		if _, err := UnmarshalTypeRestrictionRulePayload(rule.Payload); err != nil {
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

// RequiredColumnRulePayload is the payload for required column rule.
type RequiredColumnRulePayload struct {
	ColumnList []string `json:"columnList"`
}

// CommentConventionRulePayload is the payload for comment convention rule.
type CommentConventionRulePayload struct {
	Required  bool `json:"required"`
	MaxLength int  `json:"maxLength"`
}

// NumberLimitRulePayload is the payload for number limit rule.
type NumberLimitRulePayload struct {
	Number int `json:"number"`
}

// TypeRestrictionRulePayload is the payload for type restriction rule.
type TypeRestrictionRulePayload struct {
	TypeList []string `json:"typeList"`
}

// CharsetAllowlistRulePayload is the payload for charset allowlist rule.
type CharsetAllowlistRulePayload struct {
	CharsetAllowlist []string `json:"charsetAllowlist"`
}

// UnamrshalNamingRulePayloadAsRegexp will unmarshal payload to NamingRulePayload and compile it as regular expression.
func UnamrshalNamingRulePayloadAsRegexp(payload string) (*regexp.Regexp, int, error) {
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

// UnmarshalRequiredColumnRulePayload will unmarshal payload to RequiredColumnRulePayload.
func UnmarshalRequiredColumnRulePayload(payload string) (*RequiredColumnRulePayload, error) {
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

// UnmarshalNumberLimitRulePayload will unmarshal payload to NumberLimitRulePayload.
func UnmarshalNumberLimitRulePayload(payload string) (*NumberLimitRulePayload, error) {
	var nlr NumberLimitRulePayload
	if err := json.Unmarshal([]byte(payload), &nlr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal number limit rule payload %q", payload)
	}
	return &nlr, nil
}

// UnmarshalTypeRestrictionRulePayload will unmarshal payload to TypeRestrictionRulePayload.
func UnmarshalTypeRestrictionRulePayload(payload string) (*TypeRestrictionRulePayload, error) {
	var trr TypeRestrictionRulePayload
	if err := json.Unmarshal([]byte(payload), &trr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal type restriction rule payload %q", payload)
	}
	return &trr, nil
}

// UnmarshalCharsetAllowlistRulePayload will unmarshal payload to CharsetAllowlistRulePayload.
func UnmarshalCharsetAllowlistRulePayload(payload string) (*CharsetAllowlistRulePayload, error) {
	var cwr CharsetAllowlistRulePayload
	if err := json.Unmarshal([]byte(payload), &cwr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal charset allowlist rule payload %q", payload)
	}
	return &cwr, nil
}

// SQLReviewCheckContext is the context for SQL review check.
type SQLReviewCheckContext struct {
	Charset   string
	Collation string
	DbType    db.Type
	Catalog   catalog.Catalog
}

// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(statements string, ruleList []*SQLReviewRule, checkContext SQLReviewCheckContext) ([]Advice, error) {
	var result []Advice

	for _, rule := range ruleList {
		if rule.Level == SchemaRuleLevelDisabled {
			continue
		}

		advisorType, err := getAdvisorTypeByRule(rule.Type, checkContext.DbType)
		if err != nil {
			log.Printf("not supported rule: %v. error:  %v\n", rule.Type, err)
			continue
		}

		adviceList, err := Check(
			checkContext.DbType,
			advisorType,
			Context{
				Charset:   checkContext.Charset,
				Collation: checkContext.Collation,
				Rule:      rule,
				Catalog:   checkContext.Catalog.GetFinder(),
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

func getAdvisorTypeByRule(ruleType SQLReviewRuleType, engine db.Type) (Type, error) {
	switch ruleType {
	case SchemaRuleStatementRequireWhere:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLWhereRequirement, nil
		case db.Postgres:
			return PostgreSQLWhereRequirement, nil
		}
	case SchemaRuleStatementNoLeadingWildcardLike:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNoLeadingWildcardLike, nil
		case db.Postgres:
			return PostgreSQLNoLeadingWildcardLike, nil
		}
	case SchemaRuleStatementNoSelectAll:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNoSelectAll, nil
		case db.Postgres:
			return PostgreSQLNoSelectAll, nil
		}
	case SchemaRuleSchemaBackwardCompatibility:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLMigrationCompatibility, nil
		case db.Postgres:
			return PostgreSQLMigrationCompatibility, nil
		}
	case SchemaRuleTableNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingTableConvention, nil
		case db.Postgres:
			return PostgreSQLNamingTableConvention, nil
		}
	case SchemaRuleIDXNaming:
		switch engine {
		case db.MySQL, db.TiDB:
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
		case db.MySQL, db.TiDB:
			return MySQLNamingUKConvention, nil
		case db.Postgres:
			return PostgreSQLNamingUKConvention, nil
		}
	case SchemaRuleFKNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingFKConvention, nil
		case db.Postgres:
			return PostgreSQLNamingFKConvention, nil
		}
	case SchemaRuleColumnNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingColumnConvention, nil
		case db.Postgres:
			return PostgreSQLNamingColumnConvention, nil
		}
	case SchemaRuleRequiredColumn:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnRequirement, nil
		case db.Postgres:
			return PostgreSQLColumnRequirement, nil
		}
	case SchemaRuleColumnNotNull:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnNoNull, nil
		case db.Postgres:
			return PostgreSQLColumnNoNull, nil
		}
	case SchemaRuleColumnDisallowChangeType:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnDisallowChangingType, nil
		}
	case SchemaRuleColumnSetDefaultForNotNull:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnSetDefaultForNotNull, nil
		}
	case SchemaRuleColumnDisallowChange:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnDisallowChanging, nil
		}
	case SchemaRuleColumnExists:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnExists, nil
		}
	case SchemaRuleColumnDisallowChangingOrder:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnDisallowChangingOrder, nil
		}
	case SchemaRuleColumnCommentConvention:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnCommentConvention, nil
		}
	case SchemaRuleColumnAutoIncrementMustInteger:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLAutoIncrementColumnMustInteger, nil
		}
	case SchemaRuleColumnTypeRestriction:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnTypeRestriction, nil
		}
	case SchemaRuleTableRequirePK:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableRequirePK, nil
		case db.Postgres:
			return PostgreSQLTableRequirePK, nil
		}
	case SchemaRuleTableNoFK:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableNoFK, nil
		case db.Postgres:
			return PostgreSQLTableNoFK, nil
		}
	case SchemaRuleTableDropNamingConvention:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableDropNamingConvention, nil
		}
	case SchemaRuleTableExists:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableExists, nil
		}
	case SchemaRuleTableCommentConvention:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableCommentConvention, nil
		}
	case SchemaRuleMySQLEngine:
		if engine == db.MySQL {
			return MySQLUseInnoDB, nil
		}
	case SchemaRuleDropEmptyDatabase:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLDatabaseAllowDropIfEmpty, nil
		}
	case SchemaRuleIndexNoDuplicateColumn:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLIndexNoDuplicateColumn, nil
		}
	case SchemaRuleIndexKeyNumberLimit:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLIndexKeyNumberLimit, nil
		}
	case SchemaRuleStatementNoCreateTableAs:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableDisallowCreateTableAs, nil
		}
	case SchemaRuleCharsetAllowlist:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLCharsetAllowlist, nil
		}
	case SchemaRuleIndexPKType:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLIndexPKType, nil
		}
	case SchemaRuleIndexTypeNoBlob:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLIndexTypeNoBlob, nil
		}
	case SchemaRuleIndexNotExists:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLIndexNotExists, nil
		}
	}
	return Fake, errors.Errorf("unknown SQL review rule type %v for %v", ruleType, engine)
}
