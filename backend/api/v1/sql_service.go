package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/transform"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SQLService is the service for SQL.
type SQLService struct {
	v1pb.UnimplementedSQLServiceServer
}

// NewSQLService creates a SQLService.
func NewSQLService() *SQLService {
	return &SQLService{}
}

// Pretty returns pretty format SDL.
func (*SQLService) Pretty(_ context.Context, request *v1pb.PrettyRequest) (*v1pb.PrettyResponse, error) {
	engine := parser.EngineType(convertEngine(request.Engine))
	if _, err := transform.CheckFormat(engine, request.ExpectedSchema); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "User SDL is not SDL format: %s", err.Error())
	}
	if _, err := transform.CheckFormat(engine, request.CurrentSchema); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Dumped SDL is not SDL format: %s", err.Error())
	}

	prettyExpectedSchema, err := transform.SchemaTransform(engine, request.ExpectedSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to transform user SDL: %s", err.Error())
	}
	prettyCurrentSchema, err := transform.Normalize(engine, request.CurrentSchema, prettyExpectedSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to normalize dumped SDL: %s", err.Error())
	}
	return &v1pb.PrettyResponse{
		CurrentSchema:  prettyCurrentSchema,
		ExpectedSchema: prettyExpectedSchema,
	}, nil
}
