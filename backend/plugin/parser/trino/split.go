package trino

import (
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SplitSQL splits the SQL statement into a list of single SQL.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	// Trino uses semicolon (;) as statement delimiter.
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return []base.SingleSQL{}, nil
	}

	// Tokenize the statement to handle quoted strings and comments properly
	var result []base.SingleSQL
	var currentStmt strings.Builder
	var inSingleQuote, inDoubleQuote, inBlockComment bool
	var lastChar rune

	for i, char := range statement {
		currentStmt.WriteRune(char)

		// Handle string literals
		if char == '\'' && lastChar != '\\' && !inDoubleQuote && !inBlockComment {
			inSingleQuote = !inSingleQuote
		} else if char == '"' && lastChar != '\\' && !inSingleQuote && !inBlockComment {
			inDoubleQuote = !inDoubleQuote
		}

		// Handle block comments
		if char == '*' && lastChar == '/' && !inSingleQuote && !inDoubleQuote {
			inBlockComment = true
		} else if char == '/' && lastChar == '*' && inBlockComment {
			inBlockComment = false
		}

		// Split at semicolons outside of quotes and comments
		if char == ';' && !inSingleQuote && !inDoubleQuote && !inBlockComment {
			sql := currentStmt.String()
			result = append(result, base.SingleSQL{
				Text:     sql,
				BaseLine: countLines(statement[:i+1-len(sql)]),
				LastLine: countLines(statement[:i+1]),
			})
			currentStmt.Reset()
		}

		lastChar = char
	}

	// Add the last statement if it's not empty
	if currentStmt.Len() > 0 {
		sql := currentStmt.String()
		if strings.TrimSpace(sql) != "" {
			lastPos := len(statement)
			result = append(result, base.SingleSQL{
				Text:     sql,
				BaseLine: countLines(statement[:lastPos-len(sql)]),
				LastLine: countLines(statement[:lastPos]),
			})
		}
	}

	return result, nil
}

func countLines(s string) int {
	return strings.Count(s, "\n")
}

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TRINO, SplitSQL)
}
