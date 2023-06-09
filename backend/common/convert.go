package common

import (
	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	v1alpha1 "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RiskFactors are the variables when evaluating the risk level.
var RiskFactors = []cel.EnvOption{
	// string factors
	// use environment.resource_id
	cel.Variable("environment_id", cel.StringType),
	// use project.resource_id
	cel.Variable("project_id", cel.StringType),
	cel.Variable("database_name", cel.StringType),
	cel.Variable("db_engine", cel.StringType),
	cel.Variable("sql_type", cel.StringType),

	// number factors
	cel.Variable("affected_rows", cel.IntType),
}

// ApprovalFactors are the variables when finding the approval template.
var ApprovalFactors = []cel.EnvOption{
	cel.Variable("level", cel.IntType),
	cel.Variable("source", cel.IntType),
}

// ConvertParsedRisk converts parsed risk to unparsed format.
func ConvertParsedRisk(expression *v1alpha1.ParsedExpr) (*expr.Expr, error) {
	if expression == nil || expression.Expr == nil {
		return nil, nil
	}
	ast := cel.ParsedExprToAst(expression)
	expressionStr, err := cel.AstToString(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to deparse expression: %v", err)
	}
	return &expr.Expr{
		Expression: expressionStr,
	}, nil
}

// ConvertUnparsedRisk converts unparsed risk to parsed format.
func ConvertUnparsedRisk(expression *expr.Expr) (*v1alpha1.ParsedExpr, error) {
	if expression == nil || expression.Expression == "" {
		return nil, nil
	}
	e, err := cel.NewEnv(RiskFactors...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cel env")
	}

	ast, issues := e.Parse(expression.Expression)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse expression: %v", issues.Err())
	}
	expr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert ast to parsed expression: %v", err)
	}
	return expr, nil
}

// ConvertParsedApproval converts parsed approval to unparsed format.
func ConvertParsedApproval(expression *v1alpha1.ParsedExpr) (*expr.Expr, error) {
	if expression == nil || expression.Expr == nil {
		return nil, nil
	}
	ast := cel.ParsedExprToAst(expression)
	expressionStr, err := cel.AstToString(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to deparse expression: %v", err)
	}
	return &expr.Expr{
		Expression: expressionStr,
	}, nil
}

// ConvertUnparsedApproval converts unparsed approval to parsed format.
func ConvertUnparsedApproval(expression *expr.Expr) (*v1alpha1.ParsedExpr, error) {
	if expression == nil || expression.Expression == "" {
		return nil, nil
	}
	e, err := cel.NewEnv(ApprovalFactors...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cel env")
	}

	ast, issues := e.Parse(expression.Expression)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse expression: %v", issues.Err())
	}
	expr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert ast to parsed expression: %v", err)
	}
	return expr, nil
}
