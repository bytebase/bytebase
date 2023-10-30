package sqlserver

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common/log"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
)

type metricReporter struct {
	workspaceID string
	reporter    metric.Reporter
}

func (m *metricReporter) Report(metric *metric.Metric) {
	if m.workspaceID == "" {
		return
	}

	if err := m.reporter.Report(m.workspaceID, metric); err != nil {
		slog.Error(
			"Failed to report metric",
			slog.String("metric", string(metricapi.OpenAPIMetricName)),
			log.BBError(err),
		)
	}
}

func metricMiddleware(s *Server, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			requestMethod := c.Request().Method
			requestPath := c.Path()
			responseCode := c.Response().Status

			s.metricReporter.Report(&metric.Metric{
				Name:  metricapi.OpenAPIMetricName,
				Value: 1,
				Labels: map[string]any{
					"latency_ns":     strconv.FormatInt(duration.Nanoseconds(), 10),
					"request_method": requestMethod,
					"request_path":   requestPath,
					"response_code":  strconv.Itoa(responseCode),
				},
			})
		}()

		if err := next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}
