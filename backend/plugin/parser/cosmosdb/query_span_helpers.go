package cosmosdb

import (
	"log/slog"
	"strconv"

	parser "github.com/bytebase/parser/cosmosdb"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// extractFromNames returns the container name and from-clause alias.
func extractFromNames(fromClause parser.IFrom_clauseContext) (containerName, fromAlias string) {
	fromSource := fromClause.From_specification().From_source()
	containerExpr := fromSource.Container_expression()
	if containerExpr != nil && containerExpr.Container_name() != nil {
		if i := containerExpr.Container_name().Identifier(); i != nil {
			containerName = i.GetText()
		}
	}
	if i := fromSource.Identifier(); i != nil {
		fromAlias = i.GetText()
	}
	return containerName, fromAlias
}

// resolveInputAlias returns a single-element path for an input alias, resolving from-clause alias.
func resolveInputAlias(ctx parser.IScalar_expressionContext, containerName, fromAlias string) [][]base.SelectorNode {
	name := ctx.Input_alias().Identifier().GetText()
	if fromAlias != "" && name == fromAlias {
		name = containerName
	}
	return [][]base.SelectorNode{{base.NewItemSelector(name)}}
}

// buildPathAST converts a slice of SelectorNodes into a linked PathAST.
func buildPathAST(path []base.SelectorNode) *base.PathAST {
	ast := base.NewPathAST(path[0])
	next := ast.Root
	for i := 1; i < len(path); i++ {
		next.SetNext(path[i])
		next = next.GetNext()
	}
	return ast
}

// unquote strips the first and last character from a quoted string literal.
func unquote(s string) string {
	if len(s) > 1 {
		return s[1 : len(s)-1]
	}
	return s
}

// appendBracketSelector appends the bracket-access selector (string key or array index)
// to the path at index i. Shared by query_span.go and query_span_predicate.go.
func appendBracketSelector(ctx parser.IScalar_expressionContext, paths [][]base.SelectorNode, i int) {
	switch {
	case ctx.DOUBLE_QUOTE_STRING_LITERAL() != nil:
		paths[i] = append(paths[i], base.NewItemSelector(unquote(ctx.DOUBLE_QUOTE_STRING_LITERAL().GetText())))
	case ctx.SINGLE_QUOTE_STRING_LITERAL() != nil:
		paths[i] = append(paths[i], base.NewItemSelector(unquote(ctx.SINGLE_QUOTE_STRING_LITERAL().GetText())))
	case ctx.Array_index() != nil:
		if len(paths[i]) == 0 {
			return
		}
		index, err := strconv.Atoi(ctx.Array_index().GetText())
		if err != nil {
			slog.Warn("cannot convert array index to int", slog.String("index", ctx.Array_index().GetText()))
			return
		}
		last := paths[i][len(paths[i])-1]
		paths[i][len(paths[i])-1] = base.NewArraySelector(last.GetIdentifier(), index)
	default:
	}
}
