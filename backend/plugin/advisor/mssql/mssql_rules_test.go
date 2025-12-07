// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestMSSQLRules(t *testing.T) {
	mssqlRules := []storepb.SQLReviewRule_Type{
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		storepb.SQLReviewRule_NAMING_TABLE,
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		storepb.SQLReviewRule_TABLE_DISALLOW_DDL,
		storepb.SQLReviewRule_TABLE_DISALLOW_DML,
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		storepb.SQLReviewRule_COLUMN_REQUIRED,
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES,
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		storepb.SQLReviewRule_INDEX_NOT_REDUNDANT,
	}

	for _, rule := range mssqlRules {
		_, needMockData := advisorNeedMockData[rule]
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MSSQL, needMockData, false /* record */)
	}
}

// Add SQL review type here if you need metadata for test.
var advisorNeedMockData = map[storepb.SQLReviewRule_Type]bool{
	storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES: true,
	storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY:     true,
	storepb.SQLReviewRule_INDEX_NOT_REDUNDANT:               true,
}
