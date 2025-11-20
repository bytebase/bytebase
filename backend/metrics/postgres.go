package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	PostgresDatabaseSyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bytebase_postgres_metadata_sync_duration_seconds",
			Help:    "Duration of Postgres database schema synchronization",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0},
		},
		[]string{"instance", "database", "status"},
	)

	PostgresQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bytebase_postgres_metadata_query_duration_seconds",
			Help:    "Duration of individual Postgres metadata queries",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"instance", "database", "query_type", "status"},
	)
)

func init() {
	prometheus.MustRegister(PostgresDatabaseSyncDuration)
	prometheus.MustRegister(PostgresQueryDuration)
}
