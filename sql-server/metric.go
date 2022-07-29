package sqlserver

import (
	"strconv"
	"time"

	"github.com/bytebase/bytebase/common/log"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func metricMiddleware(s *Server, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			requestMethod := c.Request().Method
			requestPath := c.Path()
			responseCode := c.Response().Status

			if err := s.metricReporter.Report(&metric.Metric{
				Name:  metricAPI.OpenAPIMetricName,
				Value: 1,
				Labels: map[string]string{
					"latency_ns":     strconv.FormatInt(duration.Nanoseconds(), 10),
					"request_method": requestMethod,
					"request_path":   requestPath,
					"response_code":  strconv.Itoa(responseCode),
				},
			}); err != nil {
				log.Error(
					"Failed to report metric",
					zap.String("metric", string(metricAPI.OpenAPIMetricName)),
					zap.Error(err),
				)
			}
		}()

		if err := next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}
