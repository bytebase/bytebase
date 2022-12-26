// Package v1 is the v1 API for Bytebase.
package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	gw "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// Register registers the v1api on the grpc server and serveMux.
func Register(ctx context.Context, grpcServer *grpc.Server, mux *runtime.ServeMux, port int) error {
	gw.RegisterGreeterServiceServer(grpcServer, &GreeterServerImpl{})
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	return gw.RegisterGreeterServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf(":%d", port), opts)
}
