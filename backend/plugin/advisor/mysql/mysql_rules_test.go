package mysql

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestMySQLRules(t *testing.T) {
	for _, rule := range []storepb.SQLReviewRule_Type{
		// storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB enforce the innodb engine.
		storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB,

		// Naming related rules.
		// storepb.SQLReviewRule_NAMING_TABLE enforce the table name format.
		storepb.SQLReviewRule_NAMING_TABLE,
		// storepb.SQLReviewRule_NAMING_COLUMN enforce the column name format.
		storepb.SQLReviewRule_NAMING_COLUMN,
		// storepb.SQLReviewRule_NAMING_INDEX_UK enforce the unique key name format.
		storepb.SQLReviewRule_NAMING_INDEX_UK,
		// storepb.SQLReviewRule_NAMING_INDEX_FK enforce the foreign key name format.
		storepb.SQLReviewRule_NAMING_INDEX_FK,
		// storepb.SQLReviewRule_NAMING_INDEX_IDX enforce the index name format.
		storepb.SQLReviewRule_NAMING_INDEX_IDX,
		// storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT enforce the auto_increment column name format.
		storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT,
		// storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD enforce the identifier no keyword.
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,

		// Statement related rules.
		// storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL disallow 'SELECT *'.
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		// storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT require 'WHERE' clause for SELECT statement.
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		// storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE require 'WHERE' clause for UPDATE/DELETE statement.
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		// storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		// storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT disallow using commit in the issue.
		storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
		// storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT disallow the LIMIT clause in INSERT, DELETE and UPDATE statements.
		storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT,
		// storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY disallow the ORDER BY clause in DELETE and UPDATE statements.
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY,
		// storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE disallow redundant ALTER TABLE statements.
		storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE,
		// storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT enforce the insert row limit.
		storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
		// storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN enforce the insert column specified.
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		// storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND disallow the order by rand in the INSERT statement.
		storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
		// storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT enforce the UPDATE/DELETE affected row limit.
		storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
		// storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN dry run the dml.
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		// advisor.SchemaRuleStatementNoEqualNull disallow the equal null.
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL,
		// storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE enforce the maximum limit value.
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE,
		// storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT enforces maximum of tables in the joins.
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT,
		// advisor.SchemaRuleStatementWhereDisallowUsingFunction disallow using function in where clause.
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		// storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT enforces maximum number of logical operators in the where clause.
		storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT,
		// storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME enforce the maximum execution time.
		storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME,
		// storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION require the algorithm option in the alter table statement.
		storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION,
		// storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION require the lock option in the alter table statement.
		storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION,

		// Database related rules.
		// storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE enforce the MySQL support check if the database is empty before users drop it.
		storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE,

		// Table related rules.
		// storepb.SQLReviewRule_TABLE_REQUIRE_PK require the table to have a primary key.
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		// storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY require the table disallow the foreign key.
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		// storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION require only the table following the naming convention can be deleted.
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		// storepb.SQLReviewRule_TABLE_COMMENT enforce the table comment convention.
		storepb.SQLReviewRule_TABLE_COMMENT,
		// storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION disallow the table partition.
		storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
		// storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER disallow the table trigger.
		storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER,
		// storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX require the table no duplicate index.
		storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX,
		// storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET disallow set table charset when creating/altering table.
		storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET,

		// Column related rules.
		// storepb.SQLReviewRule_COLUMN_REQUIRED enforce the required columns in each table.
		storepb.SQLReviewRule_COLUMN_REQUIRED,
		// storepb.SQLReviewRule_COLUMN_NO_NULL enforce the columns cannot have NULL value.
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		// storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE disallow change column type.
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
		// storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL require the not null column to set default value.
		storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL,
		// storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE disallow CHANGE COLUMN statement.
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE,
		// storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER disallow changing column order.
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER,
		// storepb.SQLReviewRule_COLUMN_DISALLOW_DROPIndex disallow drop index column.
		storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX,
		// storepb.SQLReviewRule_COLUMN_COMMENT enforce the column comment convention.
		storepb.SQLReviewRule_COLUMN_COMMENT,
		// storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER require the auto-increment column to be integer.
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER,
		// storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST enforce the column type disallow list.
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		// storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET disallow set column charset.
		storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET,
		// storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH enforce the maximum character length.
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		// storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH enforce the maximum varchar length.
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		// storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE enforce the initial auto-increment value.
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE,
		// storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED enforce the auto-increment column to be unsigned.
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED,
		// storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT enforce the column default.
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,

		// Index related rules.
		// storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN require the index no duplicate column.
		storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
		// storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT enforce the index key number limit.
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		// storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT enforce the type restriction of columns in primary key.
		storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT,
		// storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB enforce the type restriction of columns in index.
		storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB,
		// storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT enforce the index total number limit.
		storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT,
		// storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST enforce the primary key type allowlist.
		storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,
		// storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST enforce the index type allowlist.
		storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST,

		// System related rules.
		// storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY enforce the MySQL and TiDB support check whether the schema change is backward compatible.
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		// storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT enforce the current column count limit.
		storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT,
		// storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST enforce the charset allowlist.
		storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
		// storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST enforce the collation allowlist.
		storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
		// advisor.SchemaRuleDisallowProcedure enforce the disallow create procedure.
		storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE,
		// advisor.SchemaRuleDisallowEvent enforce the disallow create event.
		storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE,
		// advisor.SchemaRuleDisallowView enforce the disallow create view.
		storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE,
		// storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE enforce the disallow create function.
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE,
		// storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST enforce the function disallow list.
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST,
	} {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MYSQL, false /* record */)
	}
}
