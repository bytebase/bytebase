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

func extractTableNameFromIdentifier(identifier string) string {
	if strings.HasPrefix(identifier, `"`) && strings.HasSuffix(identifier, `"`) {
		identifier = identifier[1 : len(identifier)-1]
	}
	parts := strings.Split(identifier, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
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
