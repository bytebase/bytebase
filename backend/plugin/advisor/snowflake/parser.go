// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"

	snowparser "github.com/bytebase/snowsql-parser"

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

func normalizeObjectName(objectName snowparser.IObject_nameContext) string {
	var parts []string

	if d := objectName.GetD(); d != nil {
		normalizedD := normalizeObjectNamePart(d)
		if normalizedD != "" {
			parts = append(parts, normalizedD)
		}
	}

	var schema string
	if s := objectName.GetS(); s != nil {
		normalizedS := normalizeObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	if schema == "" {
		// Backfill default schema "PUBLIC" if schema is not specified.
		schema = "PUBLIC"
	}
	parts = append(parts, schema)

	if o := objectName.GetO(); o != nil {
		normalizedO := normalizeObjectNamePart(o)
		if normalizedO != "" {
			parts = append(parts, normalizedO)
		}
	}
	return strings.Join(parts, ".")
}

func normalizeObjectNamePart(part snowparser.IId_Context) string {
	if part == nil {
		return ""
	}
	return extractOrdinaryIdentifier(part.GetText())
}

// extractOrdinaryIdentifier extracts the ordinary object name from a string. It follows the following rules:
//
// 1. If there are no double quotes on either side, it will be converted to uppercase.
//
// 2. If there are double quotes on both sides, the case will not change, the double quotes on both sides will be removed, and `""` in content will be converted to `"`.
//
// Caller MUST ensure the identifier is VALID.
func extractOrdinaryIdentifier(identifier string) string {
	quoted := strings.HasPrefix(identifier, `"`) && strings.HasSuffix(identifier, `"`)
	if quoted {
		identifier = identifier[1 : len(identifier)-1]
	}
	runeObjectName := []rune(identifier)
	var result []rune
	for i := 0; i < len(runeObjectName); i++ {
		newRune := runeObjectName[i]
		if i+1 < len(runeObjectName) && runeObjectName[i] == '"' && runeObjectName[i+1] == '"' && quoted {
			newRune = '"'
			i++
		} else if !quoted {
			newRune = unicode.ToUpper(newRune)
		}
		result = append(result, newRune)
	}
	return string(result)
}
