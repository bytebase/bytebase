// Package productmetrics collects Bytebase product-health metrics.
package productmetrics

import (
	"context"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	metricpb "github.com/prometheus/client_model/go"

	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
)

const metricsCollectionTimeout = 5 * time.Second

// Runner identifies a synchronous background control-loop cycle.
type Runner string

const (
	RunnerPlanCheck    Runner = "plan_check"
	RunnerTaskPending  Runner = "task_pending"
	RunnerTaskDispatch Runner = "task_dispatch"
	RunnerInstanceSync Runner = "instance_sync"
	RunnerDatabaseSync Runner = "database_sync"
)

// RunnerResult is the terminal result of a runner cycle.
type RunnerResult string

const (
	ResultSuccess RunnerResult = "success"
	ResultFailure RunnerResult = "failure"
	ResultSkipped RunnerResult = "skipped"
)

type databaseIdentity struct {
	instanceID   string
	databaseName string
}

type nativeHistogram struct {
	prometheus.Histogram
}

type dynamicHistogramMetric struct {
	desc        *prometheus.Desc
	histogram   prometheus.Histogram
	labelValues []string
}

type stateStore interface {
	GetWorkspaceID(context.Context) (string, error)
	CountSeatOccupyingUsers(context.Context, string) (int, error)
	CountActiveInstances(context.Context, string) (int, error)
}

type licenseStateProvider interface {
	GetVerifiedStateUncached(context.Context, string) (*enterprise.VerifiedState, error)
}

// ProductMetrics owns process-local event metrics and scrape-time product
// state. It deliberately is an unchecked Collector: dynamic resource label
// names are derived at collection time from the latest observed resource
// labels, which Prometheus's static descriptor registration cannot express.
type ProductMetrics struct {
	stores         stateStore
	licenseService licenseStateProvider

	mu sync.RWMutex

	instanceLabels     map[string]map[string]string
	databaseLabels     map[string]map[string]string
	instanceHistograms map[string]map[RunnerResult]*nativeHistogram
	databaseCounters   map[string]map[RunnerResult]uint64
	databases          map[string]databaseIdentity
	runnerEvents       map[Runner]map[RunnerResult]*nativeHistogram
}

// New creates a process-local product metrics collector.
func New(stores *store.Store, licenseService *enterprise.LicenseService) *ProductMetrics {
	metrics := &ProductMetrics{
		instanceLabels:     make(map[string]map[string]string),
		databaseLabels:     make(map[string]map[string]string),
		instanceHistograms: make(map[string]map[RunnerResult]*nativeHistogram),
		databaseCounters:   make(map[string]map[RunnerResult]uint64),
		databases:          make(map[string]databaseIdentity),
		runnerEvents:       make(map[Runner]map[RunnerResult]*nativeHistogram),
	}
	if stores != nil {
		metrics.stores = stores
	}
	if licenseService != nil {
		metrics.licenseService = licenseService
	}
	return metrics
}

// Describe intentionally sends no descriptors. Resource labels can change
// between observations, so their descriptor is defined only at collection.
func (*ProductMetrics) Describe(_ chan<- *prometheus.Desc) {
	// Dynamic descriptors are emitted only from Collect.
}

// RecordInstanceSync records one complete instance-sync attempt. Cancellation
// is a shutdown path rather than a failed synchronization.
func (m *ProductMetrics) RecordInstanceSync(instance *store.InstanceMessage, duration time.Duration, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	identity := ""
	labels := map[string]string(nil)
	if instance != nil {
		identity = instance.ResourceID
		labels = instance.Metadata.GetLabels()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instanceLabels[identity] = cloneLabels(labels)
	result := ResultSuccess
	if err != nil {
		result = ResultFailure
	}
	byResult := m.instanceHistograms[identity]
	if byResult == nil {
		byResult = make(map[RunnerResult]*nativeHistogram)
		m.instanceHistograms[identity] = byResult
	}
	h := byResult[result]
	if h == nil {
		h = newNativeHistogram()
		byResult[result] = h
	}
	h.observe(duration.Seconds())
}

// RecordDatabaseSync records one database-schema synchronization attempt.
func (m *ProductMetrics) RecordDatabaseSync(database *store.DatabaseMessage, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	identity := ""
	labels := map[string]string(nil)
	if database != nil {
		identity = database.InstanceID + "\x00" + database.DatabaseName
		labels = database.Metadata.GetLabels()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.databaseLabels[identity] = cloneLabels(labels)
	result := ResultSuccess
	if err != nil {
		result = ResultFailure
	}
	instanceID, databaseName := "", ""
	if database != nil {
		instanceID, databaseName = database.InstanceID, database.DatabaseName
	}
	m.databases[identity] = databaseIdentity{instanceID: instanceID, databaseName: databaseName}
	byResult := m.databaseCounters[identity]
	if byResult == nil {
		byResult = make(map[RunnerResult]uint64)
		m.databaseCounters[identity] = byResult
	}
	byResult[result]++
}

// RecordRunnerRun records a completed synchronous runner cycle.
func (m *ProductMetrics) RecordRunnerRun(runner Runner, result RunnerResult, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	byResult := m.runnerEvents[runner]
	if byResult == nil {
		byResult = make(map[RunnerResult]*nativeHistogram)
		m.runnerEvents[runner] = byResult
	}
	h := byResult[result]
	if h == nil {
		h = newNativeHistogram()
		byResult[result] = h
	}
	h.observe(duration.Seconds())
}

// Collect emits the process-local event state and current shared installation
// state. A state error invalidates the scrape rather than publishing a partial
// state snapshot.
func (m *ProductMetrics) Collect(ch chan<- prometheus.Metric) {
	m.collectEvents(ch)

	ctx, cancel := context.WithTimeout(context.Background(), metricsCollectionTimeout)
	defer cancel()
	if m.stores == nil || m.licenseService == nil {
		fail(ch, errors.New("product metrics dependencies are not configured"))
		return
	}
	workspaceID, err := m.stores.GetWorkspaceID(ctx)
	if err != nil {
		fail(ch, errors.Wrap(err, "failed to get workspace ID"))
		return
	}
	usedSeats, usedInstances := 0, 0
	if workspaceID != "" {
		usedSeats, err = m.stores.CountSeatOccupyingUsers(ctx, workspaceID)
		if err != nil {
			fail(ch, errors.Wrapf(err, "failed to count seat-occupying users for workspace %q", workspaceID))
			return
		}
		usedInstances, err = m.stores.CountActiveInstances(ctx, workspaceID)
		if err != nil {
			fail(ch, errors.Wrapf(err, "failed to count active instances for workspace %q", workspaceID))
			return
		}
	}
	state, err := m.licenseService.GetVerifiedStateUncached(ctx, workspaceID)
	if err != nil {
		fail(ch, errors.Wrapf(err, "failed to read verified license state for workspace %q", workspaceID))
		return
	}
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("bytebase_license_seats_used", "Number of distinct end users occupying license seats.", nil, nil), prometheus.GaugeValue, float64(usedSeats))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("bytebase_license_seats_limit", "Effective seat limit. +Inf means unlimited.", nil, nil), prometheus.GaugeValue, prometheusValue(state.UserLimit))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("bytebase_license_instances_used", "Number of non-deleted registered instances.", nil, nil), prometheus.GaugeValue, float64(usedInstances))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("bytebase_license_instances_limit", "Effective registered instance limit. +Inf means unlimited.", nil, nil), prometheus.GaugeValue, prometheusValue(state.InstanceLimit))
	expiry := math.Inf(1)
	if state.ExpiresAt != nil {
		expiry = float64(state.ExpiresAt.Unix())
	}
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("bytebase_license_expiry_timestamp_seconds", "License expiry Unix timestamp. +Inf means free or perpetual.", nil, nil), prometheus.GaugeValue, expiry)
}

func (m *ProductMetrics) collectEvents(ch chan<- prometheus.Metric) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instanceSchema := dynamicSchema(m.instanceLabels)
	instanceDesc := prometheus.NewDesc("bytebase_instance_sync_duration_seconds", "Duration of instance synchronization attempts.", append([]string{"result", "bytebase_instance_id"}, instanceSchema.keys...), nil)
	for identity, byResult := range m.instanceHistograms {
		for result, histogram := range byResult {
			labels := append([]string{string(result), identity}, instanceSchema.values(m.instanceLabels[identity])...)
			ch <- histogram.metric(instanceDesc, labels...)
		}
	}

	databaseSchema := dynamicSchema(m.databaseLabels)
	databaseDesc := prometheus.NewDesc("bytebase_database_sync_total", "Number of database schema synchronization attempts.", append([]string{"result", "bytebase_instance_id", "database"}, databaseSchema.keys...), nil)
	for identity, byResult := range m.databaseCounters {
		database := m.databases[identity]
		for result, count := range byResult {
			labels := append([]string{string(result), database.instanceID, database.databaseName}, databaseSchema.values(m.databaseLabels[identity])...)
			ch <- prometheus.MustNewConstMetric(databaseDesc, prometheus.CounterValue, float64(count), labels...)
		}
	}

	runnerDesc := prometheus.NewDesc("bytebase_runner_run_duration_seconds", "Duration of synchronous runner control-loop cycles.", []string{"runner", "result"}, nil)
	for runner, byResult := range m.runnerEvents {
		for result, histogram := range byResult {
			ch <- histogram.metric(runnerDesc, string(runner), string(result))
		}
	}
}

func newNativeHistogram() *nativeHistogram {
	return &nativeHistogram{Histogram: prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:                            "bytebase_internal_product_duration_seconds",
		Help:                            "Internal accumulator for dynamically labeled product duration metrics.",
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	})}
}

func (h *nativeHistogram) observe(value float64) {
	h.Observe(value)
}

func (h *nativeHistogram) metric(desc *prometheus.Desc, labels ...string) prometheus.Metric {
	return &dynamicHistogramMetric{desc: desc, histogram: h.Histogram, labelValues: labels}
}

func (m *dynamicHistogramMetric) Desc() *prometheus.Desc {
	return m.desc
}

func (m *dynamicHistogramMetric) Write(out *metricpb.Metric) error {
	if err := m.histogram.Write(out); err != nil {
		return errors.Wrap(err, "failed to write native histogram")
	}
	out.Label = prometheus.MakeLabelPairs(m.desc, m.labelValues)
	return nil
}

type labelSchema struct {
	keys   []string
	mapped map[string]string
}

func dynamicSchema(labelsByIdentity map[string]map[string]string) labelSchema {
	original := make(map[string]struct{})
	for _, labels := range labelsByIdentity {
		for key := range labels {
			original[key] = struct{}{}
		}
	}
	originalKeys := make([]string, 0, len(original))
	for key := range original {
		originalKeys = append(originalKeys, key)
	}
	slices.Sort(originalKeys)
	schema := labelSchema{keys: make([]string, 0, len(originalKeys)), mapped: make(map[string]string, len(originalKeys))}
	used := make(map[string]struct{})
	for _, key := range originalKeys {
		baseName := "label_" + sanitizeLabelKey(key)
		name := baseName
		for suffix := 1; ; suffix++ {
			if _, exists := used[name]; !exists {
				break
			}
			name = baseName + "_conflict" + strconv.Itoa(suffix)
		}
		used[name] = struct{}{}
		schema.keys = append(schema.keys, name)
		schema.mapped[key] = name
	}
	return schema
}

func (s labelSchema) values(labels map[string]string) []string {
	values := make([]string, len(s.keys))
	if len(labels) == 0 {
		return values
	}
	for key, value := range labels {
		name := s.mapped[key]
		for j, metricKey := range s.keys {
			if metricKey == name {
				values[j] = value
				break
			}
		}
	}
	return values
}

func sanitizeLabelKey(key string) string {
	var b strings.Builder
	for _, r := range key {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			_, _ = b.WriteRune(r)
		} else {
			_ = b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	clone := make(map[string]string, len(labels))
	for key, value := range labels {
		clone[key] = value
	}
	return clone
}

func prometheusValue(value int) float64 {
	if value == math.MaxInt {
		return math.Inf(1)
	}
	return float64(value)
}

func fail(ch chan<- prometheus.Metric, err error) {
	ch <- prometheus.NewInvalidMetric(prometheus.NewInvalidDesc(err), err)
}
