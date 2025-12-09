// Package oracle is the advisor for oracle database.
package oracle

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestOracleRules(t *testing.T) {
	for _, rule := range []storepb.SQLReviewRule_Type{
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		storepb.SQLReviewRule_NAMING_TABLE,
		storepb.SQLReviewRule_COLUMN_REQUIRED,
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE,
		storepb.SQLReviewRule_TABLE_COMMENT,
		storepb.SQLReviewRule_COLUMN_COMMENT,
	} {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_ORACLE, false /* record */)
	}
}
