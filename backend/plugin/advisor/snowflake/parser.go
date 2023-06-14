// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func parseStatement(statement string) (antlr.Tree, []advisor.Advice) {
	tree, err := parser.ParseSnowSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*parser.SyntaxError); ok {
			return nil, []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementSyntaxError,
					Title:   advisor.SyntaxErrorTitle,
					Content: syntaxErr.Message,
					Line:    syntaxErr.Line,
				},
			}
		}
		return nil, []advisor.Advice{
			{
				Status:  advisor.Warn,
				Code:    advisor.Internal,
				Title:   "Parse error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	return tree, nil
}

// extractOrdinaryIdentifier extracts the ordinary object name from a string. It follows the following rules:
//
// 1. If there are no double quotes on either side, it will be converted to uppercase.
//
// 2. If there are double quotes on both sides, the case will not change, the double quotes on both sides will be removed, and `""` in content will be converted to `"`.
//
// Caller MUST ensure the identifier is VALID.
func extractOrdinaryIdentifier(identifier string) string {
	if strings.HasPrefix(identifier, `"`) && strings.HasSuffix(identifier, `"`) {
		identifier = identifier[1 : len(identifier)-1]
	}
	runeObjectName := []rune(identifier)
	var result []rune
	for i := 0; i < len(runeObjectName); i++ {
		if i+1 < len(runeObjectName) && runeObjectName[i] == '"' && runeObjectName[i+1] == '"' {
			result = append(result, '"')
			i++
		} else {
			result = append(result, unicode.ToUpper(runeObjectName[i]))
		}
	}
	return string(result)
}

func normalizeIdentifierName(identifier string) string {
	parts := normalizeIdentifierParts(identifier)
	return strings.Join(parts, ".")
}

func normalizeIdentifierParts(identifier string) []string {
	withDoubleQuote := false
	var tidyIdentifier string
	if strings.HasPrefix(identifier, `"`) && strings.HasSuffix(identifier, `"`) {
		withDoubleQuote = true
		identifier = identifier[1 : len(identifier)-1]
		runeIdentifier := []rune(identifier)
		for i := 0; i < len(runeIdentifier); i++ {
			if runeIdentifier[i] == '"' {
				if i+1 < len(runeIdentifier) && runeIdentifier[i+1] == '"' {
					tidyIdentifier += string(runeIdentifier[i])
					i++
				}
			} else {
				tidyIdentifier += string(runeIdentifier[i])
			}
		}
	} else {
		tidyIdentifier = identifier
	}

	parts := strings.Split(tidyIdentifier, ".")
	if len(parts) == 0 {
		return []string{}
	}
	var newParts []string

	for _, part := range parts {
		if !withDoubleQuote {
			var rs []rune
			for _, r := range part {
				rs = append(rs, unicode.ToUpper(r))
			}
			newParts = append(newParts, string(rs))
		} else {
			newParts = append(newParts, part)
		}
	}
	return newParts
}
