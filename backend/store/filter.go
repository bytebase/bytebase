package store

import (
	"fmt"
	"strings"

	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
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
		case celast.CallKind:
			// Handle index selection like `labels["cost-center"]`, which is the
			// only form that survives a key CEL cannot parse as an identifier.
			if name, ok := getIndexVariable(arg); ok {
				variable = name
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

// getIndexVariable renders `labels["cost-center"]` as `labels.cost-center`, the
// same shape the dotted form produces, so callers keep one prefix to match.
// Label keys allow dashes, which CEL parses as subtraction in the dotted form,
// so index syntax is the only way such a key reaches the filter at all.
func getIndexVariable(expr celast.Expr) (string, bool) {
	call := expr.AsCall()
	if call.FunctionName() != celoperators.Index || len(call.Args()) != 2 {
		return "", false
	}
	operand, key := call.Args()[0], call.Args()[1]
	if operand.Kind() != celast.IdentKind || key.Kind() != celast.LiteralKind {
		return "", false
	}
	keyStr, ok := key.AsLiteral().Value().(string)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s.%s", operand.AsIdent(), keyStr), true
}

// buildLabelFilterSQL matches one label key against a value or a value list.
// The key is bound, never interpolated: index syntax lets it hold anything.
func buildLabelFilterSQL(resource, key string, value any) (*qb.Query, error) {
	switch v := value.(type) {
	case string:
		return qb.Q().Space(fmt.Sprintf("%s->'labels'->>?::text = ?", resource), key, v), nil
	case []any:
		if len(v) == 0 {
			return nil, errors.Errorf("empty label filter")
		}
		labelValueList := make([]any, len(v))
		for i, raw := range v {
			str, ok := raw.(string)
			if !ok {
				return nil, errors.Errorf("label value must be string, got %T", raw)
			}
			labelValueList[i] = str
		}
		return qb.Q().Space(fmt.Sprintf("%s->'labels'->>?::text = ANY(?)", resource), key, labelValueList), nil
	default:
		return nil, errors.Errorf("empty value %v for label filter", value)
	}
}

// escapeLikePattern neutralizes the wildcards in a value that is about to be
// wrapped in `%...%`. Without it a search for a statement holding `%` or `_`
// silently matches rows the user never asked for. Pair it with `ESCAPE '\'`.
func escapeLikePattern(pattern string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(pattern)
}

// containsPattern renders a substring search as an escaped LIKE pattern.
func containsPattern(value string) string {
	return "%" + escapeLikePattern(value) + "%"
}
