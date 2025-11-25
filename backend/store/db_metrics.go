package store

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// metadataDBQueryDuration tracks query latency for Bytebase's metadata database.
	metadataDBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bytebase_metadata_db_query_duration_seconds",
			Help:    "Duration of queries to Bytebase's internal metadata database",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0},
		},
		[]string{"operation", "status"},
	)
)
