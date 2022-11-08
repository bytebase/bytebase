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

// collector implements the prometheus.collector interface.
type collector struct {
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
	dbStats   = make(map[*sql.DB]stats)
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

// newStatsCollector creates a new collector.
func newStatsCollector() *collector {
	return &collector{
		maxOpenDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "max_open"),
			"Maximum number of open connections to the database.",
			[]string{"db_name", "db_type"},
			nil,
		),
		openDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "open"),
			"The number of established connections both in use and idle.",
			[]string{"db_name", "db_type"},
			nil,
		),
		inUseDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "in_use"),
			"The number of connections currently in use.",
			[]string{"db_name", "db_type"},
			nil,
		),
		idleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "idle"),
			"The number of idle connections.",
			[]string{"db_name", "db_type"},
			nil,
		),
		waitedForDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "waited_for"),
			"The total number of connections waited for.",
			[]string{"db_name", "db_type"},
			nil,
		),
		blockedSecondsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "blocked_seconds"),
			"The total time blocked waiting for a new connection.",
			[]string{"db_name", "db_type"},
			nil,
		),
		closedMaxIdleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_idle"),
			"The total number of connections closed due to SetMaxIdleConns.",
			[]string{"db_name", "db_type"},
			nil,
		),
		closedMaxLifetimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_lifetime"),
			"The total number of connections closed due to SetConnMaxLifetime.",
			[]string{"db_name", "db_type"},
			nil,
		),
		closedMaxIdleTimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_idle_time"),
			"The total number of connections closed due to SetConnMaxIdleTime.",
			[]string{"db_name", "db_type"},
			nil,
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c collector) Describe(ch chan<- *prometheus.Desc) {
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
func (c collector) Collect(ch chan<- prometheus.Metric) {
	dbStatsMu.RLock()
	defer dbStatsMu.RUnlock()
	for _, dbStats := range dbStats {
		stats := dbStats.statsFunc()
		ch <- prometheus.MustNewConstMetric(
			c.maxOpenDesc,
			prometheus.GaugeValue,
			float64(stats.MaxOpenConnections),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.openDesc,
			prometheus.GaugeValue,
			float64(stats.OpenConnections),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.inUseDesc,
			prometheus.GaugeValue,
			float64(stats.InUse),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.idleDesc,
			prometheus.GaugeValue,
			float64(stats.Idle),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.waitedForDesc,
			prometheus.CounterValue,
			float64(stats.WaitCount),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.blockedSecondsDesc,
			prometheus.CounterValue,
			stats.WaitDuration.Seconds(),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxIdleDesc,
			prometheus.CounterValue,
			float64(stats.MaxIdleClosed),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxLifetimeDesc,
			prometheus.CounterValue,
			float64(stats.MaxLifetimeClosed),
			dbStats.dbName,
			dbStats.dbType,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxIdleTimeDesc,
			prometheus.CounterValue,
			float64(stats.MaxIdleTimeClosed),
			dbStats.dbName,
			dbStats.dbType,
		)
	}
}

func init() {
	prometheus.MustRegister(newStatsCollector())
}
