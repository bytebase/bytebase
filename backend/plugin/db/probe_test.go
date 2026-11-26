package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type testEstimator struct{}

func (testEstimator) CountAffectedRows(context.Context, string) (int64, error) { return 7, nil }

type testAdapter struct{ inner []any }

func (a testAdapter) Unwrap() []any { return a.inner }

func TestAs(t *testing.T) {
	estimator, ok := As[RowEstimator](testEstimator{})
	require.True(t, ok)
	rows, err := estimator.CountAffectedRows(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, int64(7), rows)

	_, ok = As[RowEstimator](testAdapter{inner: []any{nil, "not a capability", testEstimator{}}})
	require.True(t, ok)

	_, ok = As[RowEstimator](testAdapter{})
	require.False(t, ok)
	_, ok = As[Prober](testAdapter{inner: []any{testEstimator{}}})
	require.False(t, ok)
	require.Nil(t, ProberOf(nil))
}
