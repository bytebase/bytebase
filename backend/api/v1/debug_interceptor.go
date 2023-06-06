package v1

import (
	"context"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// DebugInterceptor is the v1 debug interceptor for gRPC server.
type DebugInterceptor struct {
	errorRecordRing *api.ErrorRecordRing
}

// NewDebugInterceptor returns a new v1 API debug interceptor.
func NewDebugInterceptor(errorRecordRing *api.ErrorRecordRing) *DebugInterceptor {
	return &DebugInterceptor{
		errorRecordRing: errorRecordRing,
	}
}

// DebugInterceptor is the unary interceptor for gRPC API.
func (in *DebugInterceptor) DebugInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, request)
	if err != nil {
		in.debugInterceptorDo(ctx, request, serverInfo.FullMethod, err)
	}

	return resp, err
}

// DebugStreamInterceptor is the unary interceptor for gRPC API.
func (in *DebugInterceptor) DebugStreamInterceptor(request any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(request, ss)
	ctx := ss.Context()
	if err != nil {
		in.debugInterceptorDo(ctx, request, serverInfo.FullMethod, err)
	}

	return err
}

func (in *DebugInterceptor) debugInterceptorDo(ctx context.Context, request any, fullMethod string, err error) {
	st := status.Convert(err)
	if st.Code() == codes.Internal {
		log.Error("Internal error intercepted", zap.String("method", fullMethod), zap.Error(err), zap.Any("request", request))
	}
	if st.Code() == codes.Internal && log.EnabledLevel(zap.DebugLevel) {
		var role api.Role
		if r, ok := ctx.Value(common.RoleContextKey).(api.Role); ok {
			role = r
		}

		in.errorRecordRing.RWMutex.Lock()
		defer in.errorRecordRing.RWMutex.Unlock()
		in.errorRecordRing.Ring.Value = &v1pb.DebugLog{
			RecordTime:  timestamppb.New(time.Now()),
			RequestPath: fullMethod,
			Role:        string(role),
			Error:       err.Error(),
			StackTrace:  string(debug.Stack()),
		}
		in.errorRecordRing.Ring = in.errorRecordRing.Ring.Next()
	}
}
