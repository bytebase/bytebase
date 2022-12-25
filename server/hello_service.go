package server

import (
	"context"
	"fmt"

	gw "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// GreeterServerImpl implements the greeter service.
type GreeterServerImpl struct {
	gw.UnimplementedGreeterServiceServer
}

// Hello is the implementation of Greater Hello method.
func (*GreeterServerImpl) Hello(_ context.Context, request *gw.HelloRequest) (*gw.HelloResponse, error) {
	return &gw.HelloResponse{
		Answer: fmt.Sprintf("hello %s", request.Name),
	}, nil
}
