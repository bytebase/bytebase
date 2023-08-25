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
	cel.Variable("expiration_days", cel.IntType),
	cel.Variable("export_rows", cel.IntType),
}

// ApprovalFactors are the variables when finding the approval template.
var ApprovalFactors = []cel.EnvOption{
	cel.Variable("level", cel.IntType),
	cel.Variable("source", cel.IntType),
}

// QueryExportPolicyCELAttributes are the variables when evaluating query and export permissions.
var QueryExportPolicyCELAttributes = []cel.EnvOption{
	cel.Variable("resource.environment_name", cel.StringType),
	cel.Variable("request.time", cel.TimestampType),
	cel.Variable("request.statement", cel.StringType),
	cel.Variable("request.row_limit", cel.IntType),
	cel.Variable("resource.database", cel.StringType),
	cel.Variable("resource.schema", cel.StringType),
	cel.Variable("resource.table", cel.StringType),
}

// MaskingRulePolicyCELAttributes are the variables when evaluating masking rule.
var MaskingRulePolicyCELAttributes = []cel.EnvOption{
	cel.Variable("environment_id", cel.StringType),
	cel.Variable("project_id", cel.StringType),
	cel.Variable("instance_id", cel.StringType),
	cel.Variable("database_name", cel.StringType),
	cel.Variable("table_name", cel.StringType),
	cel.Variable("column_classification", cel.StringType),
}

// MaskingExceptionPolicyCELAttributes are the variables when evaluating masking exception.
var MaskingExceptionPolicyCELAttributes = []cel.EnvOption{
	cel.Variable("resource.instance_id", cel.StringType),
	cel.Variable("resource.database_name", cel.StringType),
	cel.Variable("resource.table_name", cel.StringType),
	cel.Variable("resource.column_name", cel.StringType),
	cel.Variable("request.time", cel.TimestampType),
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

// ValidateGroupCELExpr validates group expr.
func ValidateGroupCELExpr(expr string) (cel.Program, error) {
	e, err := cel.NewEnv(
		cel.Variable("resource", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	ast, issues := e.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, issues.Err().Error())
	}
	prog, err := e.Program(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	return prog, nil
}

// ValidateMaskingRuleCELExpr validates masking rule expr.
func ValidateMaskingRuleCELExpr(expr string) (cel.Program, error) {
	e, err := cel.NewEnv(
		MaskingRulePolicyCELAttributes...,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	ast, issues := e.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, issues.Err().Error())
	}
	prog, err := e.Program(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	return prog, nil
}

// ValidateMaskingExceptionCELExpr validates masking exception expr.
func ValidateMaskingExceptionCELExpr(expr string) (cel.Program, error) {
	e, err := cel.NewEnv(
		MaskingExceptionPolicyCELAttributes...,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	ast, issues := e.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, issues.Err().Error())
	}
	prog, err := e.Program(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	return prog, nil
}

// QueryExportFactors is the factors for query and export.
type QueryExportFactors struct {
	DatabaseNames []string
	ExportRows    int64
}

// GetQueryExportFactors is used to get risk factors from query and export expressions.
func GetQueryExportFactors(expression string) (*QueryExportFactors, error) {
	factors := &QueryExportFactors{}

	e, err := cel.NewEnv(QueryExportPolicyCELAttributes...)
	if err != nil {
		return nil, err
	}
	ast, issues := e.Compile(expression)
	if issues != nil {
		return nil, errors.Errorf("found issue %v", issues)
	}
	callExpr := ast.Expr().GetCallExpr()
	findField(callExpr, factors)
	return factors, nil
}

func findField(callExpr *v1alpha1.Expr_Call, factors *QueryExportFactors) {
	if len(callExpr.Args) == 2 {
		idExpr := callExpr.Args[0].GetIdentExpr()
		if idExpr != nil {
			if idExpr.Name == "request.row_limit" {
				factors.ExportRows = callExpr.Args[1].GetConstExpr().GetInt64Value()
			}
			if idExpr.Name == "resource.database" && callExpr.Function == "_==_" {
				factors.DatabaseNames = append(factors.DatabaseNames, callExpr.Args[1].GetConstExpr().GetStringValue())
			}
			if idExpr.Name == "resource.database" && callExpr.Function == "@in" {
				list := callExpr.Args[1].GetListExpr()
				for _, element := range list.Elements {
					factors.DatabaseNames = append(factors.DatabaseNames, element.GetConstExpr().GetStringValue())
				}
			}
			return
		}
	}
	for _, arg := range callExpr.Args {
		callExpr := arg.GetCallExpr()
		if callExpr != nil {
			findField(callExpr, factors)
		}
	}
}
