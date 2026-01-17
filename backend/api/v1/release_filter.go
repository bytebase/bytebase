package v1

import (
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"
)

// parseCategoryFilter parses CEL filter expression and extracts the category value.
// Supports: category == "value"
func parseCategoryFilter(filter string) (string, error) {
	if filter == "" {
		return "", nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return "", errors.Wrap(err, "failed to create CEL environment")
	}

	ast, iss := e.Parse(filter)
	if iss != nil {
		return "", errors.Errorf("failed to parse filter: %v", iss.String())
	}

	category, err := extractCategoryFromExpr(ast.NativeRep().Expr())
	if err != nil {
		return "", errors.Wrap(err, "failed to extract category")
	}

	return category, nil
}

// extractCategoryFromExpr walks the CEL AST to extract the category value.
func extractCategoryFromExpr(expr celast.Expr) (string, error) {
	switch expr.Kind() {
	case celast.CallKind:
		call := expr.AsCall()
		functionName := call.FunctionName()

		// Handle: category == "value"
		if functionName == celoperators.Equals {
			variable, value := getVariableAndValueFromExpr(expr)
			if variable == "category" {
				if categoryValue, ok := value.(string); ok {
					return categoryValue, nil
				}
				return "", errors.Errorf("category value must be a string, got %T", value)
			}
			return "", errors.Errorf("unsupported filter variable: %s", variable)
		}

		return "", errors.Errorf("unsupported operator: %s (only '==' is supported)", functionName)

	default:
		return "", errors.Errorf("unsupported expression type")
	}
}
