package tidb

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestTiDBRules(t *testing.T) {
	tidbRules := []storepb.SQLReviewRule_Type{
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

		// storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL disallow 'SELECT *'.
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		// storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT require 'WHERE' clause for SELECT statement.
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		// storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE require 'WHERE' clause for UPDATE and DELETE statement.
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
		// storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN enforce the insert column specified.
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		// storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND disallow the order by rand in the INSERT statement.
		storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
		// storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN dry run the dml.
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE,

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
		// storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE enforce the initial auto-increment value.
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE,
		// storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED enforce the auto-increment column to be unsigned.
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED,
		// storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT enforce the current column count limit.
		storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT,
		// storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT enforce the column default.
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,

		// storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY enforce the MySQL and TiDB support check whether the schema change is backward compatible.
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,

		// storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE enforce the MySQL and TiDB support check if the database is empty before users drop it.
		storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE,

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
		storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,

		// storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST enforce the charset allowlist.
		storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,

		// storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST enforce the collation allowlist.
		storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
	}

	for _, rule := range tidbRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_TIDB, false, false /* record */)
	}
}
