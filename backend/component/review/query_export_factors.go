package review

import (
	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	exprproto "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	"github.com/bytebase/bytebase/backend/common"
)

type queryExportFactors struct {
	Databases []string
}

func getQueryExportFactors(expression string) (*queryExportFactors, error) {
	if expression == "" {
		return &queryExportFactors{}, nil
	}

	factors := &queryExportFactors{}
	e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
	if err != nil {
		return nil, err
	}
	ast, issues := e.Compile(expression)
	if issues != nil {
		return nil, errors.Errorf("found issue %v", issues)
	}
	parsedExpr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return nil, err
	}
	databases, bounded := getDatabaseScope(parsedExpr.Expr)
	if bounded && len(databases) > 0 {
		factors.Databases = databases
	}
	return factors, nil
}

// getDatabaseScope returns a finite set covering every database the expression
// could match. Unknown predicates cannot narrow approval routing on their own.
func getDatabaseScope(expression *exprproto.Expr) ([]string, bool) {
	if constant, ok := expression.GetConstExpr().GetConstantKind().(*exprproto.Constant_BoolValue); ok && !constant.BoolValue {
		return nil, true
	}
	call := expression.GetCallExpr()
	if call == nil || len(call.Args) != 2 {
		return nil, false
	}
	switch call.Function {
	case "_&&_", "_||_":
		left, leftBounded := getDatabaseScope(call.Args[0])
		right, rightBounded := getDatabaseScope(call.Args[1])
		if call.Function == "_||_" {
			if !leftBounded || !rightBounded {
				return nil, false
			}
			return common.Uniq(append(left, right...)), true
		}
		if !leftBounded {
			return right, rightBounded
		}
		if !rightBounded {
			return left, true
		}
		rightSet := make(map[string]bool, len(right))
		for _, database := range right {
			rightSet[database] = true
		}
		intersection := make([]string, 0)
		for _, database := range left {
			if rightSet[database] {
				intersection = append(intersection, database)
			}
		}
		return intersection, true
	case "_==_":
		for i, operand := range call.Args {
			if operand.GetIdentExpr().GetName() != common.CELAttributeResourceDatabase {
				continue
			}
			if value, ok := call.Args[1-i].GetConstExpr().GetConstantKind().(*exprproto.Constant_StringValue); ok {
				return []string{value.StringValue}, true
			}
		}
	case "@in":
		if call.Args[0].GetIdentExpr().GetName() != common.CELAttributeResourceDatabase {
			return nil, false
		}
		list := call.Args[1].GetListExpr()
		if list == nil {
			return nil, false
		}
		databases := make([]string, 0, len(list.Elements))
		for _, element := range list.Elements {
			value, ok := element.GetConstExpr().GetConstantKind().(*exprproto.Constant_StringValue)
			if !ok {
				return nil, false
			}
			databases = append(databases, value.StringValue)
		}
		return common.Uniq(databases), true
	default:
		return nil, false
	}
	return nil, false
}
