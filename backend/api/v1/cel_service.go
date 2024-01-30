package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/cel-go/cel"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// CelService represents a service for managing CEL.
type CelService struct {
	v1pb.UnimplementedCelServiceServer
}

// NewCelService returns a CEL service instance.
func NewCelService() *CelService {
	return &CelService{}
}

// Parse parses a CEL expression.
func (*CelService) BatchParse(_ context.Context, request *v1pb.BatchParseRequest) (*v1pb.BatchParseResponse, error) {
	e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CEL environment: %v", err)
	}
	resp := &v1pb.BatchParseResponse{}
	for _, expression := range request.Expressions {
		ast, issues := e.Parse(expression)
		if issues != nil && issues.Err() != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to parse expression: %v", issues.Err())
		}
		expr, err := cel.AstToParsedExpr(ast)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert ast to parsed expression: %v", err)
		}
		resp.Expressions = append(resp.Expressions, expr)
	}
	return resp, nil
}

// Deparse deparses a parsed CEL expression.
func (*CelService) BatchDeparse(_ context.Context, request *v1pb.BatchDeparseRequest) (*v1pb.BatchDeparseResponse, error) {
	resp := &v1pb.BatchDeparseResponse{}
	for _, expression := range request.Expressions {
		ast := cel.ParsedExprToAst(expression)
		expressionStr, err := cel.AstToString(ast)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to deparse expression: %v", err)
		}
		resp.Expressions = append(resp.Expressions, expressionStr)
	}
	return resp, nil
}
