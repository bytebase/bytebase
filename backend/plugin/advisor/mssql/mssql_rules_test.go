// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

func TestMSSQLRules(t *testing.T) {
	snowflakeRules := []advisor.SQLReviewRuleType{
		advisor.SchemaRuleStatementNoSelectAll,
	}

	for _, rule := range snowflakeRules {
		advisor.RunSQLReviewRuleTest(t, rule, db.MSSQL, false /* record */)
	}
}
