// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

func TestOracleRules(t *testing.T) {
	rules := []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_NAMING_TABLE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^[a-z]+(_[a-z]+)*$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRED, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"id", "created_ts", "updated_ts", "creator_id", "updater_id"}}}},
		{Type: storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"JSON", "BINARY_FLOAT"}}}},
		{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 20}}},
		{Type: storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_COLUMN_NO_NULL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 2560}}},
		{Type: storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingCasePayload{NamingCasePayload: &storepb.SQLReviewRule_NamingCaseRulePayload{Upper: true}}},
		{Type: storepb.SQLReviewRule_TABLE_COMMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_CommentConventionPayload{CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true, MaxLength: 10}}},
		{Type: storepb.SQLReviewRule_COLUMN_COMMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_CommentConventionPayload{CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true, MaxLength: 10}}},
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, Level: storepb.SQLReviewRule_WARNING},
	}

	for _, rule := range rules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_ORACLE, false /* record */)
	}
}

func TestOracleAdvisorUsesOmniWithoutANTLRFallback(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		rule      *storepb.SQLReviewRule
		advisor   advisor.Advisor
		wantCount int
	}{
		{
			name:      "select no select all",
			statement: "SELECT * FROM users",
			rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
				Level: storepb.SQLReviewRule_WARNING,
			},
			advisor:   &SelectNoSelectAllAdvisor{},
			wantCount: 1,
		},
		{
			name:      "update inline view target",
			statement: "UPDATE (SELECT * FROM tech_book WHERE UPPER(name) = 'X') v SET v.creator = 'y'",
			rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
				Level: storepb.SQLReviewRule_WARNING,
			},
			advisor:   &StatementWhereDisallowFunctionsAndCalculationsAdvisor{},
			wantCount: 1,
		},
		{
			name:      "delete inline view target",
			statement: "DELETE FROM (SELECT * FROM tech_book WHERE ABS(id) > 5) v",
			rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
				Level: storepb.SQLReviewRule_WARNING,
			},
			advisor:   &StatementWhereDisallowFunctionsAndCalculationsAdvisor{},
			wantCount: 1,
		},
		{
			name: "plsql loop nested dml",
			statement: `BEGIN
  FOR i IN 1..2 LOOP
    UPDATE tech_book SET creator = 'x' WHERE UPPER(name) = 'A';
  END LOOP;
END;`,
			rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
				Level: storepb.SQLReviewRule_WARNING,
			},
			advisor:   &StatementWhereDisallowFunctionsAndCalculationsAdvisor{},
			wantCount: 1,
		},
		{
			name: "plsql case nested dml",
			statement: `BEGIN
  CASE
    WHEN 1 = 1 THEN
      DELETE FROM tech_book WHERE ABS(id) > 5;
  END CASE;
END;`,
			rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
				Level: storepb.SQLReviewRule_WARNING,
			},
			advisor:   &StatementWhereDisallowFunctionsAndCalculationsAdvisor{},
			wantCount: 1,
		},
		{
			name:      "json column type",
			statement: "CREATE TABLE t(a int, b JSON)",
			rule: &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
				Level: storepb.SQLReviewRule_WARNING,
				Payload: &storepb.SQLReviewRule_StringArrayPayload{
					StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{
						List: []string{"JSON"},
					},
				},
			},
			advisor:   &ColumnTypeDisallowListAdvisor{},
			wantCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_ORACLE, test.statement)
			if err != nil {
				t.Fatal(err)
			}

			adviceList, err := test.advisor.Check(context.Background(), advisor.Context{
				Rule:                  test.rule,
				DBType:                storepb.Engine_ORACLE,
				CurrentDatabase:       "TEST_DB",
				DBSchema:              advisor.MockOracleDatabase,
				IsObjectCaseSensitive: true,
				ParsedStatements:      stmts,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(adviceList) != test.wantCount {
				t.Fatalf("got %d advices, want %d", len(adviceList), test.wantCount)
			}

			assertOracleStmtsDidNotUseANTLRFallback(t, stmts)
		})
	}
}

func assertOracleStmtsDidNotUseANTLRFallback(t *testing.T, stmts []base.ParsedStatement) {
	t.Helper()
	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		if _, ok := base.GetANTLRAST(stmt.AST); ok {
			t.Fatalf("Oracle advisor used ANTLR fallback for %T", stmt.AST)
		}
	}
}
