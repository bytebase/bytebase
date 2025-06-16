package v1

import (
	"context"

	"connectrpc.com/connect"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/cel-go/cel"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
)

// CelService represents a service for managing CEL.
type CelService struct {
	v1connect.UnimplementedCelServiceHandler
}

// NewCelService returns a CEL service instance.
func NewCelService() *CelService {
	return &CelService{}
}

// BatchParse parses a CEL expression.
func (*CelService) BatchParse(
	_ context.Context,
	req *connect.Request[v1pb.BatchParseRequest],
) (*connect.Response[v1pb.BatchParseResponse], error) {
	e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CEL environment: %v", err)
	}
	resp := &v1pb.BatchParseResponse{}
	for _, expression := range req.Msg.Expressions {
		ast, issues := e.Parse(expression)
		if issues != nil && issues.Err() != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to parse expression: %v", issues.Err())
		}
		parsedExpr, err := cel.AstToParsedExpr(ast)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert ast to parsed expression: %v", err)
		}
		resp.Expressions = append(resp.Expressions, parsedExpr.Expr)
	}
	return connect.NewResponse(resp), nil
}

// BatchDeparse deparses a parsed CEL expression.
func (*CelService) BatchDeparse(
	_ context.Context,
	req *connect.Request[v1pb.BatchDeparseRequest],
) (*connect.Response[v1pb.BatchDeparseResponse], error) {
	resp := &v1pb.BatchDeparseResponse{}
	for _, expression := range req.Msg.Expressions {
		ast := cel.ParsedExprToAst(&expr.ParsedExpr{Expr: expression})
		expressionStr, err := cel.AstToString(ast)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to deparse expression: %v", err)
		}
		resp.Expressions = append(resp.Expressions, expressionStr)
	}
	return connect.NewResponse(resp), nil
}
