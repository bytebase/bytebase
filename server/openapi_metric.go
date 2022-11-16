package server

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
)

func openAPIMetricMiddleware(s *Server, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			requestMethod := c.Request().Method
			requestPath := c.Path()
			responseCode := c.Response().Status

			if s.MetricReporter != nil {
				s.MetricReporter.report(&metric.Metric{
					Name:  metricAPI.OpenAPIMetricName,
					Value: 1,
					Labels: map[string]interface{}{
						"latency_ns":     strconv.FormatInt(duration.Nanoseconds(), 10),
						"request_method": requestMethod,
						"request_path":   requestPath,
						"response_code":  strconv.Itoa(responseCode),
					},
				})
			}
		}()

		if err := next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}
