package parser

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
)

// ValidateSQLForEditor validates the SQL statement for editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE
// 2. SELECT statement
//
// We also support CTE with SELECT statements, but not with DML statements.
func ValidateSQLForEditor(engine EngineType, statement string) bool {
	switch engine {
	case Postgres, Redshift:
		return postgresValidateSQLForEditor(statement)
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
		log.Debug("Failed to remove quoted text and comment", zap.String("statement", statement), zap.Error(err))
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
		log.Debug("Failed to remove quoted text and comment", zap.String("statement", statement), zap.Error(err))
		return false
	}

	return checkStatementWithoutQuotedTextAndComment(textWithoutQuotedAndComment)
}

// postgresValidateSQLForEditor validates the SQL statement for SQL editor.
// Consider that the tokenizer cannot handle the dollar-sign($), so that we use pg_query_go to parse the statement.
// For EXPLAIN and normal SELECT statements, we can directly use regexp to check.
// For CTE, we need to parse the statement to JSON and check the JSON keys.
func postgresValidateSQLForEditor(statement string) bool {
	jsonText, err := pgquery.ParseToJSON(statement)
	if err != nil {
		log.Debug("Failed to parse statement to JSON", zap.String("statement", statement), zap.Error(err))
		return false
	}

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
		var jsonData map[string]any

		if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
			log.Debug("Failed to unmarshal JSON", zap.String("jsonText", jsonText), zap.Error(err))
			return false
		}

		dmlKeyList := []string{"InsertStmt", "UpdateStmt", "DeleteStmt"}

		return !keyExistsInJSONData(jsonData, dmlKeyList)
	}

	return false
}

func keyExistsInJSONData(jsonData map[string]any, keyList []string) bool {
	for _, key := range keyList {
		if _, ok := jsonData[key]; ok {
			return true
		}
	}

	for _, value := range jsonData {
		switch v := value.(type) {
		case map[string]any:
			if keyExistsInJSONData(v, keyList) {
				return true
			}
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					if keyExistsInJSONData(m, keyList) {
						return true
					}
				}
			}
		}
	}

	return false
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
	case Postgres:
		return "", errors.Errorf("unsupported engine type: %s", engine)
	case MySQL, TiDB, MariaDB, OceanBase:
		return mysqlRemoveQuotedTextAndComment(statement)
	case Standard, Oracle, MSSQL:
		return standardRemoveQuotedTextAndComment(statement)
	}
	return "", errors.Errorf("unsupported engine type: %s", engine)
}

func standardRemoveQuotedTextAndComment(statement string) (string, error) {
	var buf bytes.Buffer
	t := newTokenizer(statement)

	t.skipBlank()
	startPos := t.pos()
	for {
		switch {
		case t.char(0) == '/' && t.char(1) == '*':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '-' && t.char(1) == '-':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '\'':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanString('\''); err != nil {
				return "'", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '"':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanIdentifier('"'); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == eofRune:
			text := t.getString(startPos, t.pos()-startPos)
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			return buf.String(), nil
		default:
			t.skip(1)
		}
	}
}

func mysqlRemoveQuotedTextAndComment(statement string) (string, error) {
	var buf bytes.Buffer
	t := newTokenizer(statement)

	t.skipBlank()
	startPos := t.pos()
	for {
		switch {
		case t.char(0) == eofRune:
			text := t.getString(startPos, t.pos()-startPos)
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			return buf.String(), nil
		case t.char(0) == '/' && t.char(1) == '*':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '-' && t.char(1) == '-':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '#':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '\'' || t.char(0) == '"':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanString(t.char(0)); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '`':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanIdentifier('`'); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		default:
			t.skip(1)
		}
	}
}
