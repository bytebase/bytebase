package util

import (
	"database/sql"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "go_sql_stats"
	subsystem = "connections"
)

type statsFunc func() sql.DBStats

type stats struct {
	dbType    string
	dbName    string
	statsFunc statsFunc
}

// Collector implements the prometheus.Collector interface.
type Collector struct {
	maxOpenDesc           *prometheus.Desc
	openDesc              *prometheus.Desc
	inUseDesc             *prometheus.Desc
	idleDesc              *prometheus.Desc
	waitedForDesc         *prometheus.Desc
	blockedSecondsDesc    *prometheus.Desc
	closedMaxIdleDesc     *prometheus.Desc
	closedMaxLifetimeDesc *prometheus.Desc
	closedMaxIdleTimeDesc *prometheus.Desc
}

var (
	dbStatsMu sync.RWMutex
	dbStats   map[*sql.DB]stats
)

// RegisterStats register dbStats metrics.
func RegisterStats(dbType, dbName string, db *sql.DB) {
	dbStatsMu.Lock()
	defer dbStatsMu.Unlock()
	dbStats[db] = stats{
		dbType:    dbType,
		dbName:    dbName,
		statsFunc: db.Stats,
	}
}

// UnregisterStats unregister dbStats metrics.
func UnregisterStats(db *sql.DB) {
	dbStatsMu.Lock()
	defer dbStatsMu.Unlock()
	delete(dbStats, db)
}

// NewStatsCollector creates a new collector.
func NewStatsCollector() *Collector {
	return &Collector{
		maxOpenDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "max_open"),
			"Maximum number of open connections to the database.",
			nil,
			nil,
		),
		openDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "open"),
			"The number of established connections both in use and idle.",
			nil,
			nil,
		),
		inUseDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "in_use"),
			"The number of connections currently in use.",
			nil,
			nil,
		),
		idleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "idle"),
			"The number of idle connections.",
			nil,
			nil,
		),
		waitedForDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "waited_for"),
			"The total number of connections waited for.",
			nil,
			nil,
		),
		blockedSecondsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "blocked_seconds"),
			"The total time blocked waiting for a new connection.",
			nil,
			nil,
		),
		closedMaxIdleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_idle"),
			"The total number of connections closed due to SetMaxIdleConns.",
			nil,
			nil,
		),
		closedMaxLifetimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_lifetime"),
			"The total number of connections closed due to SetConnMaxLifetime.",
			nil,
			nil,
		),
		closedMaxIdleTimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_idle_time"),
			"The total number of connections closed due to SetConnMaxIdleTime.",
			nil,
			nil,
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpenDesc
	ch <- c.openDesc
	ch <- c.inUseDesc
	ch <- c.idleDesc
	ch <- c.waitedForDesc
	ch <- c.blockedSecondsDesc
	ch <- c.closedMaxIdleDesc
	ch <- c.closedMaxLifetimeDesc
	ch <- c.closedMaxIdleTimeDesc
}

// Collect implements the prometheus.Collector interface.
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	dbStatsMu.RLock()
	defer dbStatsMu.RUnlock()
	for _, dbStats := range dbStats {
		stats := dbStats.statsFunc()
		ch <- prometheus.MustNewConstMetric(
			c.maxOpenDesc,
			prometheus.GaugeValue,
			float64(stats.MaxOpenConnections),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.openDesc,
			prometheus.GaugeValue,
			float64(stats.OpenConnections),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.inUseDesc,
			prometheus.GaugeValue,
			float64(stats.InUse),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.idleDesc,
			prometheus.GaugeValue,
			float64(stats.Idle),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.waitedForDesc,
			prometheus.CounterValue,
			float64(stats.WaitCount),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.blockedSecondsDesc,
			prometheus.CounterValue,
			stats.WaitDuration.Seconds(),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxIdleDesc,
			prometheus.CounterValue,
			float64(stats.MaxIdleClosed),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxLifetimeDesc,
			prometheus.CounterValue,
			float64(stats.MaxLifetimeClosed),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxIdleTimeDesc,
			prometheus.CounterValue,
			float64(stats.MaxIdleTimeClosed),
			"db_name",
			dbStats.dbName,
			"db_type",
			dbStats.dbType,
		)
	}
}

func init() {
	prometheus.MustRegister(NewStatsCollector())
}
