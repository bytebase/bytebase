package store

import (
	"fmt"

	celast "github.com/google/cel-go/common/ast"
)

func getVariableAndValueFromExpr(expr celast.Expr) (string, any) {
	var variable string
	var value any
	for _, arg := range expr.AsCall().Args() {
		switch arg.Kind() {
		case celast.IdentKind:
			variable = arg.AsIdent()
		case celast.SelectKind:
			// Handle member selection like "labels.environment"
			sel := arg.AsSelect()
			if sel.Operand().Kind() == celast.IdentKind {
				variable = fmt.Sprintf("%s.%s", sel.Operand().AsIdent(), sel.FieldName())
			}
		case celast.LiteralKind:
			value = arg.AsLiteral().Value()
		case celast.ListKind:
			list := []any{}
			for _, e := range arg.AsList().Elements() {
				if e.Kind() == celast.LiteralKind {
					list = append(list, e.AsLiteral().Value())
				}
			}
			value = list
		default:
		}
	}
	return variable, value
}
