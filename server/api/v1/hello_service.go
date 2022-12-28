package v1

import (
	"context"
	"fmt"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// GreeterServerImpl implements the greeter service.
type GreeterServerImpl struct {
	v1pb.UnimplementedGreeterServiceServer
}

// Hello is the implementation of Greater Hello method.
func (*GreeterServerImpl) Hello(_ context.Context, request *v1pb.HelloRequest) (*v1pb.HelloResponse, error) {
	return &v1pb.HelloResponse{
		Answer: fmt.Sprintf("hello %s", request.Name),
	}, nil
}
