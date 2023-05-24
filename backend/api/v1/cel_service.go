package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/cel-go/cel"

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

var queryAttributes = []cel.EnvOption{
	cel.Variable("request.time", cel.TimestampType),
	cel.Variable("resource.database", cel.StringType),
	cel.Variable("request.statement", cel.StringType),
	cel.Variable("request.row_limit", cel.IntType),
	cel.Variable("request.export_format", cel.StringType),
}

func (*CelService) Parse(ctx context.Context, request *v1pb.ParseRequest) (*v1pb.ParseResponse, error) {
	e, err := cel.NewEnv(queryAttributes...)
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

func (*CelService) Deparse(ctx context.Context, request *v1pb.DeparseRequest) (*v1pb.DeparseResponse, error) {
	ast := cel.ParsedExprToAst(request.Expression)
	expressionStr, err := cel.AstToString(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to deparse expression: %v", err)
	}
	return &v1pb.DeparseResponse{
		Expression: expressionStr,
	}, nil
}
