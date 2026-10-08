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
	callExpr := parsedExpr.Expr.GetCallExpr()
	findField(callExpr, factors)
	return factors, nil
}

func findField(callExpr *exprproto.Expr_Call, factors *queryExportFactors) {
	if callExpr == nil {
		return
	}
	if len(callExpr.Args) == 2 {
		idExpr := callExpr.Args[0].GetIdentExpr()
		if idExpr != nil {
			if idExpr.Name == common.CELAttributeResourceDatabase && callExpr.Function == "_==_" {
				factors.Databases = append(factors.Databases, callExpr.Args[1].GetConstExpr().GetStringValue())
			}
			if idExpr.Name == common.CELAttributeResourceDatabase && callExpr.Function == "@in" {
				list := callExpr.Args[1].GetListExpr()
				for _, element := range list.Elements {
					factors.Databases = append(factors.Databases, element.GetConstExpr().GetStringValue())
				}
			}
			return
		}
	}
	for _, arg := range callExpr.Args {
		callExpr := arg.GetCallExpr()
		findField(callExpr, factors)
	}
}
