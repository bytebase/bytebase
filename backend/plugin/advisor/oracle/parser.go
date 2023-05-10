// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func parseStatement(statement string) (antlr.Tree, []advisor.Advice) {
	tree, err := parser.ParsePLSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*parser.SyntaxError); ok {
			return nil, []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.StatementSyntaxError,
					Title:   advisor.SyntaxErrorTitle,
					Content: syntaxErr.Message,
					Line:    syntaxErr.Line,
				},
			}
		}
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.Internal,
				Title:   "Parse error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	return tree, nil
}

func normalizeIdentifier(ctx antlr.ParserRuleContext, currentSchema string) string {
	switch ctx := ctx.(type) {
	case *plsql.IdentifierContext:
		return normalizeIdentifierContext(ctx)
	case *plsql.Schema_nameContext:
		return normalizeIdentifierContext(ctx.Identifier())
	case *plsql.Table_nameContext:
		return normalizeIdentifierContext(ctx.Identifier())
	case *plsql.Tableview_nameContext:
		if ctx.Identifier() != nil {
			result := []string{normalizeIdentifierContext(ctx.Identifier())}
			if ctx.Id_expression() != nil {
				result = append(result, normalizeIDExpression(ctx.Id_expression()))
			}
			if len(result) == 1 {
				result = []string{currentSchema, result[0]}
			}
			return strings.Join(result, ".")
		}
		return ""
	case *plsql.Column_nameContext:
		result := []string{normalizeIdentifierContext(ctx.Identifier())}
		for _, idExpression := range ctx.AllId_expression() {
			result = append(result, normalizeIDExpression(idExpression))
		}
		return strings.Join(result, ".")
	}
	return ""
}

func normalizeIdentifierContext(identifier plsql.IIdentifierContext) string {
	if identifier == nil {
		return ""
	}

	return normalizeIDExpression(identifier.Id_expression())
}

func normalizeIDExpression(idExpression plsql.IId_expressionContext) string {
	if idExpression == nil {
		return ""
	}

	regularID := idExpression.Regular_id()
	if regularID != nil {
		return strings.ToUpper(regularID.GetText())
	}

	delimitedID := idExpression.DELIMITED_ID()
	if delimitedID != nil {
		return strings.Trim(delimitedID.GetText(), "\"")
	}

	return ""
}

func normalizeTableName(name string) string {
	list := strings.Split(name, ".")
	var result []string
	for _, item := range list {
		result = append(result, fmt.Sprintf("\"%s\"", item))
	}
	return strings.Join(result, ".")
}

func lastIdentifier(name string) string {
	list := strings.Split(name, ".")
	return list[len(list)-1]
}
