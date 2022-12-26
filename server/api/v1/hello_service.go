package v1

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/common"
	gw "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// GreeterServerImpl implements the greeter service.
type GreeterServerImpl struct {
	gw.UnimplementedGreeterServiceServer
}

// Hello is the implementation of Greater Hello method.
func (*GreeterServerImpl) Hello(ctx context.Context, request *gw.HelloRequest) (*gw.HelloResponse, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	return &gw.HelloResponse{
		Answer: fmt.Sprintf("hello %s from %v", request.Name, principalID),
	}, nil
}
