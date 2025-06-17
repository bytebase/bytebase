package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/pkg/errors"

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

func (in *DebugInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		startTime := time.Now()
		resp, err := next(ctx, req)
		in.debugInterceptorDo(ctx, req.Spec().Procedure, err, startTime)

		// Truncate error message to 10240 characters.
		if connectErr := (&connect.Error{}); errors.As(err, &connectErr) {
			if msg, truncated := common.TruncateString(connectErr.Message(), 10240); truncated {
				slog.Info("Truncated error message", slog.String("fullMethod", req.Spec().Procedure), slog.String("original error message", connectErr.Message()))
				err = connect.NewError(connectErr.Code(), errors.New("[TRUNCATED] "+msg))
			}
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
	connectErr := (&connect.Error{})
	if !errors.As(err, &connectErr) {
		connectErr = connect.NewError(connect.CodeUnknown, err)
	}
	var logLevel slog.Level
	var logMsg string
	switch connectErr.Code() {
	case connect.CodeUnauthenticated, connect.CodeOutOfRange, connect.CodePermissionDenied, connect.CodeNotFound, connect.CodeInvalidArgument:
		logLevel = slog.LevelDebug
		logMsg = "client error"
	case connect.CodeInternal, connect.CodeUnknown, connect.CodeDataLoss, connect.CodeUnavailable, connect.CodeDeadlineExceeded:
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
