package parser

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

// ValidateSQLForEditor validates the SQL statement for editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE
// 2. SELECT statement
//
// We also support CTE with SELECT statements, but not with DML statements.
func ValidateSQLForEditor(engine EngineType, statement string) bool {
	switch engine {
	case Postgres, Redshift, RisingWave:
		return pgparser.ValidateSQLForEditor(statement)
	case MySQL, TiDB, MariaDB, OceanBase:
		return mysqlValidateSQLForEditor(statement)
	default:
		return standardValidateSQLForEditor(statement)
	}
}

// standardValidateSQLForEditor validates the SQL statement for SQL editor.
// We validate the statement by following steps:
// 1. Remove all quoted text(quoted identifier, string literal) and comments from the statement.
// 2. Use regexp to check if the statement is a normal SELECT statement and EXPLAIN statement.
// 3. For CTE, use regexp to check if the statement has UPDATE, DELETE and INSERT statements.
func standardValidateSQLForEditor(statement string) bool {
	textWithoutQuotedAndComment, err := removeQuotedTextAndComment(Standard, statement)
	if err != nil {
		slog.Debug("Failed to remove quoted text and comment", slog.String("statement", statement), log.BBError(err))
		return false
	}

	return checkStatementWithoutQuotedTextAndComment(textWithoutQuotedAndComment)
}

// mysqlValidateSQLForEditor validates the SQL statement for SQL editor.
// We validate the statement by following steps:
// 1. Remove all quoted text(quoted identifier, string literal) and comments from the statement.
// 2. Use regexp to check if the statement is a normal SELECT statement and EXPLAIN statement.
// 3. For CTE, use regexp to check if the statement has UPDATE, DELETE and INSERT statements.
func mysqlValidateSQLForEditor(statement string) bool {
	textWithoutQuotedAndComment, err := removeQuotedTextAndComment(MySQL, statement)
	if err != nil {
		slog.Debug("Failed to remove quoted text and comment", slog.String("statement", statement), log.BBError(err))
		return false
	}

	return checkStatementWithoutQuotedTextAndComment(textWithoutQuotedAndComment)
}

func checkStatementWithoutQuotedTextAndComment(statement string) bool {
	formattedStr := strings.ToUpper(strings.TrimSpace(statement))
	if isSelect, _ := regexp.MatchString(`^SELECT\s+?`, formattedStr); isSelect {
		return true
	}

	if isSelect, _ := regexp.MatchString(`^SELECT\*\s+?`, formattedStr); isSelect {
		return true
	}

	if isExplain, _ := regexp.MatchString(`^EXPLAIN\s+?`, formattedStr); isExplain {
		if isExplainAnalyze, _ := regexp.MatchString(`^EXPLAIN\s+ANALYZE\s+?`, formattedStr); isExplainAnalyze {
			return false
		}
		return true
	}

	cteRegex := regexp.MustCompile(`^WITH\s+?`)
	if matchResult := cteRegex.MatchString(formattedStr); matchResult {
		dmlRegs := []string{`\bINSERT\b`, `\bUPDATE\b`, `\bDELETE\b`}
		for _, reg := range dmlRegs {
			if matchResult, _ := regexp.MatchString(reg, formattedStr); matchResult {
				return false
			}
		}
		return true
	}

	return false
}

func removeQuotedTextAndComment(engine EngineType, statement string) (string, error) {
	switch engine {
	case Postgres, RisingWave:
		return "", errors.Errorf("unsupported engine type: %s", engine)
	case MySQL, TiDB, MariaDB, OceanBase:
		return tokenizer.MysqlRemoveQuotedTextAndComment(statement)
	case Standard, Oracle, MSSQL:
		return tokenizer.StandardRemoveQuotedTextAndComment(statement)
	}
	return "", errors.Errorf("unsupported engine type: %s", engine)
}
