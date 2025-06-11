package v1

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
)

// ActuatorServiceConnectHandler implements the Connect RPC interface for ActuatorService.
type ActuatorServiceConnectHandler struct {
	v1connect.UnimplementedActuatorServiceHandler
	service *ActuatorService
}

// NewActuatorServiceConnectHandler creates a new Connect RPC handler for ActuatorService.
func NewActuatorServiceConnectHandler(service *ActuatorService) *ActuatorServiceConnectHandler {
	return &ActuatorServiceConnectHandler{
		service: service,
	}
}

// GetActuatorInfo gets the actuator info.
func (h *ActuatorServiceConnectHandler) GetActuatorInfo(
	ctx context.Context,
	req *connect.Request[v1pb.GetActuatorInfoRequest],
) (*connect.Response[v1pb.ActuatorInfo], error) {
	info, err := h.service.GetActuatorInfo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}

// UpdateActuatorInfo updates the actuator info.
func (h *ActuatorServiceConnectHandler) UpdateActuatorInfo(
	ctx context.Context,
	req *connect.Request[v1pb.UpdateActuatorInfoRequest],
) (*connect.Response[v1pb.ActuatorInfo], error) {
	info, err := h.service.UpdateActuatorInfo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}

// SetupSample sets up the sample project and instance.
func (h *ActuatorServiceConnectHandler) SetupSample(
	ctx context.Context,
	req *connect.Request[v1pb.SetupSampleRequest],
) (*connect.Response[emptypb.Empty], error) {
	empty, err := h.service.SetupSample(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(empty), nil
}

// DeleteCache deletes the cache.
func (h *ActuatorServiceConnectHandler) DeleteCache(
	ctx context.Context,
	req *connect.Request[v1pb.DeleteCacheRequest],
) (*connect.Response[emptypb.Empty], error) {
	empty, err := h.service.DeleteCache(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(empty), nil
}

// GetResourcePackage gets the theme resources.
func (h *ActuatorServiceConnectHandler) GetResourcePackage(
	ctx context.Context,
	req *connect.Request[v1pb.GetResourcePackageRequest],
) (*connect.Response[v1pb.ResourcePackage], error) {
	pkg, err := h.service.GetResourcePackage(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(pkg), nil
}
