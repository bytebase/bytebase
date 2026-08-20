package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// newMetricsHandler keeps the server-local registry isolated while folding in
// package-global instrumentation that is registered elsewhere in Bytebase.
func newMetricsHandler(registry *prometheus.Registry) http.Handler {
	return promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(
			prometheus.Gatherers{registry, prometheus.DefaultGatherer},
			promhttp.HandlerOpts{ErrorHandling: promhttp.HTTPErrorOnError},
		),
	)
}
