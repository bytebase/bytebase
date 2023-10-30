package v1

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/metric"
	metricplugin "github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// DebugInterceptor is the v1 debug interceptor for gRPC server.
type DebugInterceptor struct {
	errorRecordRing *api.ErrorRecordRing
	profile         *config.Profile
	metricReporter  *metricreport.Reporter
}

// NewDebugInterceptor returns a new v1 API debug interceptor.
func NewDebugInterceptor(errorRecordRing *api.ErrorRecordRing, profile *config.Profile, metricReporter *metricreport.Reporter) *DebugInterceptor {
	return &DebugInterceptor{
		errorRecordRing: errorRecordRing,
		profile:         profile,
		metricReporter:  metricReporter,
	}
}

// DebugInterceptor is the unary interceptor for gRPC API.
func (in *DebugInterceptor) DebugInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	startTime := time.Now()
	resp, err := handler(ctx, request)
	in.debugInterceptorDo(ctx, serverInfo.FullMethod, err, startTime)

	// Truncate error message to 1024 characters.
	st, _ := status.FromError(err)
	if msg, truncated := common.TruncateString(st.Message(), 1024); truncated {
		slog.Info("Truncated error message", slog.String("fullMethod", serverInfo.FullMethod), slog.String("original error message", st.Message()))
		stp := st.Proto()
		stp.Message = "[TRUNCATED] " + msg
		err = status.FromProto(stp).Err()
	}

	return resp, err
}

// DebugStreamInterceptor is the unary interceptor for gRPC API.
func (in *DebugInterceptor) DebugStreamInterceptor(request any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	startTime := time.Now()
	err := handler(request, ss)
	ctx := ss.Context()
	in.debugInterceptorDo(ctx, serverInfo.FullMethod, err, startTime)

	return err
}

func (in *DebugInterceptor) debugInterceptorDo(ctx context.Context, fullMethod string, err error, startTime time.Time) {
	st := status.Convert(err)
	var logLevel slog.Level
	var logMsg string
	switch st.Code() {
	case codes.OK:
		logLevel = slog.LevelDebug
		logMsg = "OK"
	case codes.Unauthenticated, codes.OutOfRange, codes.PermissionDenied, codes.NotFound:
		logLevel = slog.LevelDebug
		logMsg = "client error"
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unavailable, codes.DeadlineExceeded:
		logLevel = slog.LevelError
		logMsg = "server error"
	default:
		logLevel = slog.LevelError
		logMsg = "unknown error"
	}
	slog.Log(ctx, logLevel, logMsg, "method", fullMethod, log.BBError(err), "latency", fmt.Sprintf("%vms", time.Since(startTime).Milliseconds()))
	if st.Code() == codes.Internal && slog.Default().Enabled(ctx, slog.LevelDebug) {
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
	if _, ok := ctx.Value(common.PrincipalIDContextKey).(int); ok {
		// Only update for authorized request.
		in.profile.LastActiveTs = time.Now().Unix()
	}
	in.metricReporter.Report(ctx, &metricplugin.Metric{
		Name:  metric.APIRequestMetricName,
		Value: 1,
		Labels: map[string]any{
			"method": fullMethod,
		},
	})
}
