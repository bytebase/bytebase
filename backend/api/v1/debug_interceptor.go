package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/metric"
	metricplugin "github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
)

// DebugInterceptor is the v1 debug interceptor for gRPC server.
type DebugInterceptor struct {
	metricReporter *metricreport.Reporter
}

// NewDebugInterceptor returns a new v1 API debug interceptor.
func NewDebugInterceptor(metricReporter *metricreport.Reporter) *DebugInterceptor {
	return &DebugInterceptor{
		metricReporter: metricReporter,
	}
}

// DebugInterceptor is the unary interceptor for gRPC API.
func (in *DebugInterceptor) DebugInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	startTime := time.Now()
	resp, err := handler(ctx, request)
	in.debugInterceptorDo(ctx, serverInfo.FullMethod, err, startTime)

	// Truncate error message to 10240 characters.
	st, _ := status.FromError(err)
	if msg, truncated := common.TruncateString(st.Message(), 10240); truncated {
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

func (in *DebugInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		startTime := time.Now()
		resp, err := next(ctx, req)
		in.debugInterceptorDo(ctx, req.Spec().Procedure, err, startTime)

		// Truncate error message to 10240 characters.
		st, _ := status.FromError(err)
		if msg, truncated := common.TruncateString(st.Message(), 10240); truncated {
			slog.Info("Truncated error message", slog.String("fullMethod", req.Spec().Procedure), slog.String("original error message", st.Message()))
			stp := st.Proto()
			stp.Message = "[TRUNCATED] " + msg
			err = status.FromProto(stp).Err()
		}

		return resp, err
	}
}

func (*DebugInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}
func (in *DebugInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		startTime := time.Now()
		err := next(ctx, conn)
		in.debugInterceptorDo(ctx, conn.Spec().Procedure, err, startTime)

		return err
	}
}

func (in *DebugInterceptor) debugInterceptorDo(ctx context.Context, fullMethod string, err error, startTime time.Time) {
	st := status.Convert(err)
	var logLevel slog.Level
	var logMsg string
	switch st.Code() {
	case codes.OK:
		logLevel = slog.LevelDebug
		logMsg = "OK"
	case codes.Unauthenticated, codes.OutOfRange, codes.PermissionDenied, codes.NotFound, codes.InvalidArgument:
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
	in.metricReporter.Report(ctx, &metricplugin.Metric{
		Name:  metric.APIRequestMetricName,
		Value: 1,
		Labels: map[string]any{
			"method": fullMethod,
		},
	})
}
