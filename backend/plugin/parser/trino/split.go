package trino

import (
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TRINO, SplitSQL)
}

// SplitSQL splits the SQL statement into a list of single SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	// Trino uses semicolon (;) as statement delimiter.
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return []base.SingleSQL{}, nil
	}

	// Tokenize the statement to handle quoted strings and comments properly
	var result []base.SingleSQL
	var currentStmt strings.Builder
	var inSingleQuote, inDoubleQuote, inBlockComment, inLineComment bool
	var lastChar rune

	for i, char := range statement {
		// Handle line comments
		if char == '\n' && inLineComment {
			inLineComment = false
		}

		// Only add character to current statement if not in a line comment
		if !inLineComment {
			currentStmt.WriteRune(char)
		}

		// Handle string literals (only if we're not in a comment)
		if !inBlockComment && !inLineComment {
			if char == '\'' && lastChar != '\\' && !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else if char == '"' && lastChar != '\\' && !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}

		// Handle comments (only if we're not in a string literal)
		if !inSingleQuote && !inDoubleQuote {
			// Line comments
			if char == '-' && lastChar == '-' && !inBlockComment && !inLineComment {
				inLineComment = true
			}
			// Block comments
			if char == '*' && lastChar == '/' && !inLineComment {
				inBlockComment = true
			} else if char == '/' && lastChar == '*' && inBlockComment {
				inBlockComment = false
			}
		}

		// Split at semicolons outside of quotes and comments
		if char == ';' && !inSingleQuote && !inDoubleQuote && !inBlockComment && !inLineComment {
			sql := currentStmt.String()
			result = append(result, base.SingleSQL{
				Text:     sql,
				BaseLine: countLines(statement[:i+1-len(sql)]),
				End: &storepb.Position{
					Line: int32(countLines(statement[:i+1])),
				},
			})
			currentStmt.Reset()
		}

		lastChar = char
	}

	// Add the last statement if it's not empty
	if currentStmt.Len() > 0 {
		sql := currentStmt.String()
		if strings.TrimSpace(sql) != "" {
			result = append(result, base.SingleSQL{
				Text:     sql,
				BaseLine: countLines(statement[:len(statement)-len(sql)]),
				End: &storepb.Position{
					Line: int32(countLines(statement)),
				},
			})
		}
	}

	return result, nil
}

// countLines counts the number of newlines in a string
func countLines(s string) int {
	return strings.Count(s, "\n")
}
