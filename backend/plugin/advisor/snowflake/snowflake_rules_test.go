// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestSnowflakeRules(t *testing.T) {
	rules := []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_NAMING_TABLE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^[A-Z]+(_[A-Z]+)*$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 2560}}},
		{Type: storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRED, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"id", "created_ts", "updated_ts", "creator_id", "updater_id"}}}},
		{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingCasePayload{NamingCasePayload: &storepb.SQLReviewRule_NamingCaseRulePayload{Upper: true}}},
		{Type: storepb.SQLReviewRule_COLUMN_NO_NULL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "_delete$"}}},
		{Type: storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, Level: storepb.SQLReviewRule_WARNING},
	}

	for _, rule := range rules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_SNOWFLAKE, false /* record */)
	}
}
