package productmetrics

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metricpb "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type fakeStateStore struct {
	workspace string
	seats     int
	instances int
	err       error
}

func (s *fakeStateStore) GetWorkspaceID(context.Context) (string, error) {
	return s.workspace, s.err
}

func (s *fakeStateStore) CountSeatOccupyingUsers(context.Context, string) (int, error) {
	return s.seats, s.err
}

func (s *fakeStateStore) CountActiveInstances(context.Context, string) (int, error) {
	return s.instances, s.err
}

type fakeLicenseStateProvider struct {
	state *enterprise.VerifiedState
	err   error
}

func (p *fakeLicenseStateProvider) GetVerifiedStateUncached(context.Context, string) (*enterprise.VerifiedState, error) {
	return p.state, p.err
}

func TestRegistryInstallationStateVariants(t *testing.T) {
	finiteExpiry := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	expiredAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	tests := []struct {
		name       string
		state      *enterprise.VerifiedState
		wantExpiry float64
		wantLimit  float64
	}{
		{name: "free", state: &enterprise.VerifiedState{UserLimit: 20, InstanceLimit: 10}, wantExpiry: math.Inf(1), wantLimit: 10},
		{name: "perpetual", state: &enterprise.VerifiedState{UserLimit: math.MaxInt, InstanceLimit: math.MaxInt}, wantExpiry: math.Inf(1), wantLimit: math.Inf(1)},
		{name: "finite", state: &enterprise.VerifiedState{ExpiresAt: &finiteExpiry, UserLimit: 100, InstanceLimit: 25}, wantExpiry: float64(finiteExpiry.Unix()), wantLimit: 25},
		{name: "expired", state: &enterprise.VerifiedState{ExpiresAt: &expiredAt, UserLimit: 20, InstanceLimit: 10}, wantExpiry: float64(expiredAt.Unix()), wantLimit: 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metrics := New(nil, nil)
			metrics.stores = &fakeStateStore{workspace: "ws", seats: 3, instances: 2}
			metrics.licenseService = &fakeLicenseStateProvider{state: test.state}
			registry := prometheus.NewRegistry()
			registry.MustRegister(metrics)
			families, err := registry.Gather()
			require.NoError(t, err)
			require.Equal(t, 2.0, familyGaugeValue(t, families, "bytebase_license_instances_used"))
			require.Equal(t, test.wantLimit, familyGaugeValue(t, families, "bytebase_license_instances_limit"))
			require.Equal(t, test.wantExpiry, familyGaugeValue(t, families, "bytebase_license_expiry_timestamp_seconds"))
		})
	}
}

func TestRegistryStateFailureAndEventLocality(t *testing.T) {
	newCollector := func() (*ProductMetrics, *prometheus.Registry) {
		metrics := New(nil, nil)
		metrics.stores = &fakeStateStore{workspace: "ws"}
		metrics.licenseService = &fakeLicenseStateProvider{state: &enterprise.VerifiedState{UserLimit: 20, InstanceLimit: 10}}
		registry := prometheus.NewRegistry()
		registry.MustRegister(metrics)
		return metrics, registry
	}
	metrics1, registry1 := newCollector()
	_, registry2 := newCollector()
	metrics1.RecordRunnerRun(RunnerPlanCheck, ResultSuccess, time.Second)
	families1, err := registry1.Gather()
	require.NoError(t, err)
	families2, err := registry2.Gather()
	require.NoError(t, err)
	require.Equal(t, 1.0, familyHistogramCount(families1, "bytebase_runner_run_duration_seconds"))
	require.Zero(t, familyHistogramCount(families2, "bytebase_runner_run_duration_seconds"))

	metrics1.licenseService = &fakeLicenseStateProvider{err: errors.New("malformed license")}
	_, err = registry1.Gather()
	require.ErrorContains(t, err, "malformed license")
}

func TestResourceMetricLabelsAndCumulativeMutation(t *testing.T) {
	m := New(nil, nil)
	instance := &store.InstanceMessage{
		ResourceID: "prod",
		Metadata: &storepb.Instance{Labels: map[string]string{
			"env.prod":     "before",
			"env-prod":     "dash",
			"\u4e2d\u6587": "unicode",
		}},
	}
	m.RecordInstanceSync(instance, time.Second, nil)
	collidingMetric := findMetric(t, eventMetrics(m), "bytebase_instance_sync_duration_seconds")
	require.Equal(t, "before", metricLabel(collidingMetric, "label_env_prod_conflict1"))
	require.Equal(t, "dash", metricLabel(collidingMetric, "label_env_prod"))
	require.Equal(t, "unicode", metricLabel(collidingMetric, "label___"))
	m.RecordInstanceSync(&store.InstanceMessage{ResourceID: "staging", Metadata: &storepb.Instance{Labels: map[string]string{"owner": "platform"}}}, time.Second, nil)
	unionMetric := findMetricWithLabels(t, eventMetrics(m), "bytebase_instance_sync_duration_seconds", map[string]string{"bytebase_instance_id": "staging"})
	require.Equal(t, "", metricLabel(unionMetric, "label_env_prod_conflict1"))
	require.Equal(t, "platform", metricLabel(unionMetric, "label_owner"))
	instance.Metadata.Labels = map[string]string{"env.prod": "after"}
	m.RecordInstanceSync(instance, 2*time.Second, nil)

	metrics := eventMetrics(m)
	metric := findMetric(t, metrics, "bytebase_instance_sync_duration_seconds")
	require.Equal(t, uint64(2), metric.GetHistogram().GetSampleCount())
	require.Equal(t, 3.0, metric.GetHistogram().GetSampleSum())
	require.Equal(t, int32(3), metric.GetHistogram().GetSchema())
	require.Empty(t, metric.GetHistogram().GetBucket())
	require.Equal(t, "prod", metricLabel(metric, "bytebase_instance_id"))
	require.Equal(t, "after", metricLabel(metric, "label_env_prod"))
}

func TestDatabaseAndRunnerMetrics(t *testing.T) {
	m := New(nil, nil)
	database := &store.DatabaseMessage{
		InstanceID:   "prod",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{"region": "us"}},
	}
	m.RecordDatabaseSync(database, nil)
	m.RecordDatabaseSync(database, context.DeadlineExceeded)
	m.RecordDatabaseSync(database, context.Canceled)
	m.RecordRunnerRun(RunnerPlanCheck, ResultSuccess, time.Second)

	metrics := eventMetrics(m)
	databaseMetric := findMetricWithLabels(t, metrics, "bytebase_database_sync_total", map[string]string{"result": "success"})
	require.Equal(t, "prod", metricLabel(databaseMetric, "bytebase_instance_id"))
	require.Equal(t, "app", metricLabel(databaseMetric, "database"))
	require.Equal(t, "success", metricLabel(databaseMetric, "result"))
	require.Equal(t, 1.0, databaseMetric.GetCounter().GetValue())
	runnerMetric := findMetric(t, metrics, "bytebase_runner_run_duration_seconds")
	require.Equal(t, int32(3), runnerMetric.GetHistogram().GetSchema())
	require.Empty(t, runnerMetric.GetHistogram().GetBucket())
}

func TestNativeHistogramBucketCap(t *testing.T) {
	histogram := newNativeHistogram()
	for i := range 150 {
		histogram.observe(math.Pow(2, float64(i)))
	}
	require.LessOrEqual(t, len(histogram.buckets), 100)
	require.Less(t, histogram.schema, int32(3))
}

func TestConcurrentObserveAndCollect(t *testing.T) {
	m := New(nil, nil)
	instance := &store.InstanceMessage{ResourceID: "prod", Metadata: &storepb.Instance{}}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 100 {
				m.RecordInstanceSync(instance, time.Millisecond, nil)
				eventMetrics(m)
			}
		})
	}
	wg.Wait()
	metric := findMetric(t, eventMetrics(m), "bytebase_instance_sync_duration_seconds")
	require.Equal(t, uint64(1000), metric.GetHistogram().GetSampleCount())
}

func eventMetrics(m *ProductMetrics) []*metricpb.Metric {
	ch := make(chan prometheus.Metric, 1000)
	m.collectEvents(ch)
	metrics := make([]*metricpb.Metric, 0, len(ch))
	for range len(ch) {
		metric := <-ch
		written := &metricpb.Metric{}
		if metric.Write(written) == nil {
			metrics = append(metrics, written)
		}
	}
	return metrics
}

func findMetric(t *testing.T, metrics []*metricpb.Metric, name string) *metricpb.Metric {
	return findMetricWithLabels(t, metrics, name, nil)
}

func findMetricWithLabels(t *testing.T, metrics []*metricpb.Metric, name string, labels map[string]string) *metricpb.Metric {
	t.Helper()
	for _, metric := range metrics {
		matches := true
		for name, value := range labels {
			matches = matches && metricLabel(metric, name) == value
		}
		if !matches {
			continue
		}
		if metric.GetHistogram() != nil && name == "bytebase_instance_sync_duration_seconds" && metricLabel(metric, "bytebase_instance_id") != "" {
			return metric
		}
		if metric.GetCounter() != nil && name == "bytebase_database_sync_total" {
			return metric
		}
		if metric.GetHistogram() != nil && name == "bytebase_runner_run_duration_seconds" && metricLabel(metric, "runner") != "" {
			return metric
		}
	}
	t.Fatalf("metric %q not found", name)
	return nil
}

func metricLabel(metric *metricpb.Metric, name string) string {
	for _, pair := range metric.GetLabel() {
		if pair.GetName() == name {
			return pair.GetValue()
		}
	}
	return ""
}

func familyGaugeValue(t *testing.T, families []*metricpb.MetricFamily, name string) float64 {
	t.Helper()
	for _, family := range families {
		if family.GetName() == name {
			require.Len(t, family.Metric, 1)
			return family.Metric[0].GetGauge().GetValue()
		}
	}
	t.Fatalf("metric family %q not found", name)
	return 0
}

func familyHistogramCount(families []*metricpb.MetricFamily, name string) float64 {
	for _, family := range families {
		if family.GetName() == name && len(family.Metric) > 0 {
			return float64(family.Metric[0].GetHistogram().GetSampleCount())
		}
	}
	return 0
}
