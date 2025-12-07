package pg

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestPostgreSQLANTLRRules(t *testing.T) {
	antlrRules := []storepb.SQLReviewRule_Type{
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
		storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
		storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
		storepb.SQLReviewRule_COLUMN_COMMENT,
		storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH,
		storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY,
		storepb.SQLReviewRule_NAMING_INDEX_FK,
		storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED,
		storepb.SQLReviewRule_NAMING_INDEX_IDX,
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
		storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,
		storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT,
		storepb.SQLReviewRule_NAMING_INDEX_PK,
		storepb.SQLReviewRule_COLUMN_REQUIRED,
		storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE,
		storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE,
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		storepb.SQLReviewRule_NAMING_COLUMN,
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID,
		storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID,
		storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE,
		storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE,
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL,
		storepb.SQLReviewRule_TABLE_COMMENT,
		storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		storepb.SQLReviewRule_NAMING_TABLE,
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		storepb.SQLReviewRule_NAMING_INDEX_UK,
	}

	for _, rule := range antlrRules {
		RunPGSQLReviewRuleTest(t, rule, storepb.Engine_POSTGRES, false /* record */)
	}
}
