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
	instance  string
	engine    string
	env       string
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
func RegisterStats(env, instance, engine string, db *sql.DB) {
	dbStatsMu.Lock()
	defer dbStatsMu.Unlock()
	dbStats[db] = stats{
		instance:  instance,
		env:       env,
		engine:    engine,
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
			[]string{"env", "instance", "engine"},
			nil,
		),
		openDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "open"),
			"The number of established connections both in use and idle.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		inUseDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "in_use"),
			"The number of connections currently in use.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		idleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "idle"),
			"The number of idle connections.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		waitedForDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "waited_for"),
			"The total number of connections waited for.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		blockedSecondsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "blocked_seconds"),
			"The total time blocked waiting for a new connection.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		closedMaxIdleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_idle"),
			"The total number of connections closed due to SetMaxIdleConns.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		closedMaxLifetimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_lifetime"),
			"The total number of connections closed due to SetConnMaxLifetime.",
			[]string{"env", "instance", "engine"},
			nil,
		),
		closedMaxIdleTimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "closed_max_idle_time"),
			"The total number of connections closed due to SetConnMaxIdleTime.",
			[]string{"env", "instance", "engine"},
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
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.openDesc,
			prometheus.GaugeValue,
			float64(stats.OpenConnections),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.inUseDesc,
			prometheus.GaugeValue,
			float64(stats.InUse),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.idleDesc,
			prometheus.GaugeValue,
			float64(stats.Idle),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.waitedForDesc,
			prometheus.CounterValue,
			float64(stats.WaitCount),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.blockedSecondsDesc,
			prometheus.CounterValue,
			stats.WaitDuration.Seconds(),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxIdleDesc,
			prometheus.CounterValue,
			float64(stats.MaxIdleClosed),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxLifetimeDesc,
			prometheus.CounterValue,
			float64(stats.MaxLifetimeClosed),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
		ch <- prometheus.MustNewConstMetric(
			c.closedMaxIdleTimeDesc,
			prometheus.CounterValue,
			float64(stats.MaxIdleTimeClosed),
			dbStats.env,
			dbStats.instance,
			dbStats.engine,
		)
	}
}

func init() {
	prometheus.MustRegister(newStatsCollector())
}
