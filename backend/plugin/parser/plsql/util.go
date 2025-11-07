package plsql

import (
	"github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"
)

func plsqlNormalizeColumnName(currentSchema string, ctx plsql.IColumn_nameContext) (string, string, string, error) {
	var buf []string
	buf = append(buf, NormalizeIdentifierContext(ctx.Identifier()))
	for _, idExpression := range ctx.AllId_expression() {
		buf = append(buf, NormalizeIDExpression(idExpression))
	}
	switch len(buf) {
	case 1:
		return currentSchema, "", buf[0], nil
	case 2:
		return currentSchema, buf[0], buf[1], nil
	case 3:
		return buf[0], buf[1], buf[2], nil
	default:
		return "", "", "", errors.Errorf("invalid column name: %s", ctx.GetText())
	}
}

func normalizeColumnAlias(ctx plsql.IColumn_aliasContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Identifier() != nil {
		return NormalizeIdentifierContext(ctx.Identifier())
	}

	if ctx.Quoted_string() != nil {
		return ctx.Quoted_string().GetText()
	}

	return ""
}

func NormalizeTableAlias(ctx plsql.ITable_aliasContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Identifier() != nil {
		return NormalizeIdentifierContext(ctx.Identifier())
	}

	if ctx.Quoted_string() != nil {
		return ctx.Quoted_string().GetText()
	}

	return ""
}

// NormalizeQuotedString returns the string without the quotes.
func NormalizeQuotedString(ctx plsql.IQuoted_stringContext) string {
	if ctx == nil {
		return ""
	}

	raw := ctx.GetText()
	return raw[1 : len(raw)-1]
}

func normalizeClusterName(ctx plsql.ICluster_nameContext) (string, string) {
	var list []string
	for _, idExpression := range ctx.AllId_expression() {
		list = append(list, NormalizeIDExpression(idExpression))
	}

	switch len(list) {
	case 1:
		return "", list[0]
	case 2:
		return list[0], list[1]
	default:
		return "", ""
	}
}
