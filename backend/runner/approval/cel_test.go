package approval

import (
	"fmt"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func TestValidateSQLSelectStatement(t *testing.T) {
	env, _ := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("request.time", decls.String, nil),
			decls.NewIdent("request.export_format", decls.String, nil),
			decls.NewIdent("request.row_limit", decls.Int, nil),
			decls.NewIdent("resource.database", decls.String, nil),
			decls.NewIdent("resource.schema", decls.String, nil),
			decls.NewIdent("resource.table", decls.NewListType(decls.String), nil),
		),
	)

	src := `request.time < timestamp("2023-06-30T10:03:13.412Z") && request.export_format == "CSV" && request.row_limit == 1000 && (resource.database == "instances/ins-nzq-bohu/databases/employee" && resource.schema == "" && resource.table in ["department"])`
	ast, err := env.Compile(src)
	if err != nil {
		fmt.Println("Failed to compile the expression:", err)
		return
	}

	// 提取操作符后面的值
	values := []interface{}{}
	extractValues(ast.Expr(), &values)

	fmt.Println("Values:", values)
}

func extractValueFromExpr(e *expr.Expr) interface{} {
	if e == nil {
		return nil
	}

	switch e.GetExprKind().(type) {
	case *expr.Expr_CallExpr:
		callExpr := e.GetCallExpr()
		if callExpr == nil || len(callExpr.Args) == 0 {
			return nil
		}

		// 递归提取操作符后面的值
		return extractValueFromExpr(callExpr.Args[len(callExpr.Args)-1])
	case *expr.Expr_ConstExpr:
		constExpr := e.GetConstExpr()
		if constExpr == nil {
			return nil
		}

		switch constVal := constExpr.ConstantKind.(type) {
		case *expr.Constant_BoolValue:
			return constVal.BoolValue
		case *expr.Constant_BytesValue:
			return string(constVal.BytesValue)
		case *expr.Constant_DoubleValue:
			return constVal.DoubleValue
		case *expr.Constant_Int64Value:
			return constVal.Int64Value
		case *expr.Constant_StringValue:
			return constVal.StringValue
		case *expr.Constant_Uint64Value:
			return constVal.Uint64Value
		}
	}

	return nil
}

func extractValues(e *expr.Expr, values *[]interface{}) {
	if e == nil {
		return
	}

	switch e.GetExprKind().(type) {
	case *expr.Expr_CallExpr:
		callExpr := e.GetCallExpr()
		if callExpr == nil {
			return
		}

		for _, arg := range callExpr.Args {
			extractValues(arg, values)
		}
	case *expr.Expr_ConstExpr:
		value := extractValueFromExpr(e)
		if value != nil {
			*values = append(*values, value)
		}
	}
}
