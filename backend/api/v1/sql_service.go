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
func (s *SQLService) Pretty(_ context.Context, request *v1pb.PrettyRequest) (*v1pb.PrettyResponse, error) {
	engine := parser.EngineType(convertEngine(request.Engine))
	if _, err := transform.CheckFormat(engine, request.UserSDL); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "User SDL is not SDL format: %s", err.Error())
	}
	if _, err := transform.CheckFormat(engine, request.DumpedSDL); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Dumped SDL is not SDL format: %s", err.Error())
	}

	prettyUserSDL, err := transform.SchemaTransform(engine, request.UserSDL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to transform user SDL: %s", err.Error())
	}
	prettyDumpedSDL, err := transform.Normalize(engine, request.DumpedSDL, prettyUserSDL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to normalize dumped SDL: %s", err.Error())
	}
	return &v1pb.PrettyResponse{
		PrettyDumpedSDL: prettyDumpedSDL,
		PrettyUserSDL:   prettyUserSDL,
	}, nil
}
