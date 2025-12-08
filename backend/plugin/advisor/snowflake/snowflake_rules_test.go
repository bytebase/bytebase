// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestSnowflakeRules(t *testing.T) {
	snowflakeRules := []storepb.SQLReviewRule_Type{
		storepb.SQLReviewRule_NAMING_TABLE,
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		storepb.SQLReviewRule_COLUMN_REQUIRED,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE,
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
	}

	for _, rule := range snowflakeRules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_SNOWFLAKE, false, false /* record */)
	}
}
