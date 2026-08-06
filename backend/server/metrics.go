package server

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
)

// metricsCollectionTimeout bounds each collection so a slow metadata read cannot
// hang a scrape indefinitely. Collectors receive no request context.
const metricsCollectionTimeout = 5 * time.Second

// licenseSeatsCollector exposes product-level license seat gauges at /metrics.
// Both values are computed synchronously on every scrape from fresh shared
// metadata (no per-replica caches), so replicas expose equivalent values.
// A failed metadata read fails the whole scrape instead of emitting zero,
// stale, or missing gauges.
type licenseSeatsCollector struct {
	stores         *store.Store
	licenseService *enterprise.LicenseService

	usedSeats *prometheus.Desc
	seatLimit *prometheus.Desc
}

func newLicenseSeatsCollector(stores *store.Store, licenseService *enterprise.LicenseService) *licenseSeatsCollector {
	return &licenseSeatsCollector{
		stores:         stores,
		licenseService: licenseService,
		usedSeats: prometheus.NewDesc(
			"bytebase_license_seats_used",
			"Number of distinct end users occupying license seats in this Bytebase installation, including pending invites and group members, excluding deleted users and non-user identities.",
			nil, nil,
		),
		seatLimit: prometheus.NewDesc(
			"bytebase_license_seats_limit",
			"Effective seat limit enforced for this Bytebase installation. +Inf means the license has no seat limit.",
			nil, nil,
		),
	}
}

func (c *licenseSeatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usedSeats
	ch <- c.seatLimit
}

func (c *licenseSeatsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), metricsCollectionTimeout)
	defer cancel()

	workspaceID, err := c.stores.GetWorkspaceID(ctx)
	if err != nil {
		failMetricCollection(ch, errors.Wrap(err, "failed to get workspace ID"))
		return
	}

	used := 0
	if workspaceID != "" {
		used, err = c.stores.CountSeatOccupyingUsers(ctx, workspaceID)
		if err != nil {
			failMetricCollection(ch, errors.Wrapf(err, "failed to count seat-occupying users for workspace %q", workspaceID))
			return
		}
	}

	limit, err := c.licenseService.GetUserLimitUncached(ctx, workspaceID)
	if err != nil {
		failMetricCollection(ch, errors.Wrapf(err, "failed to read the user limit for workspace %q", workspaceID))
		return
	}

	limitValue := float64(limit)
	if limit == math.MaxInt {
		limitValue = math.Inf(1)
	}

	ch <- prometheus.MustNewConstMetric(c.usedSeats, prometheus.GaugeValue, float64(used))
	ch <- prometheus.MustNewConstMetric(c.seatLimit, prometheus.GaugeValue, limitValue)
}

// failMetricCollection emits an invalid metric, which makes Registry.Gather
// fail and therefore fails the whole scrape (HTTP 500 via HTTPErrorOnError)
// instead of reporting zero, stale, or partial values.
func failMetricCollection(ch chan<- prometheus.Metric, err error) {
	ch <- prometheus.NewInvalidMetric(prometheus.NewInvalidDesc(err), err)
}
