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
func (*CelService) Parse(_ context.Context, request *v1pb.ParseRequest) (*v1pb.ParseResponse, error) {
	e, err := cel.NewEnv(common.QueryExportPolicyCELAttributes...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CEL environment: %v", err)
	}
	ast, issues := e.Parse(request.Expression)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse expression: %v", issues.Err())
	}
	expr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert ast to parsed expression: %v", err)
	}
	return &v1pb.ParseResponse{
		Expression: expr,
	}, nil
}

// Deparse deparses a parsed CEL expression.
func (*CelService) Deparse(_ context.Context, request *v1pb.DeparseRequest) (*v1pb.DeparseResponse, error) {
	ast := cel.ParsedExprToAst(request.Expression)
	expressionStr, err := cel.AstToString(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to deparse expression: %v", err)
	}
	return &v1pb.DeparseResponse{
		Expression: expressionStr,
	}, nil
}
