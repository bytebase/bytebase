package clickhouse

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranslateAggregateFunctionError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.NoError(t, translateAggregateFunctionError(nil))
	})

	t.Run("non aggregate function error", func(t *testing.T) {
		err := errors.New("some other clickhouse error")
		require.Same(t, err, translateAggregateFunctionError(err))
	})

	t.Run("simple aggregate function error is not rewritten", func(t *testing.T) {
		// Defensive: SimpleAggregateFunction is supported by the driver, but if a
		// future driver bug ever surfaces an "unsupported column type" error for
		// it, our marker must not match (because SimpleAggregateFunction errors
		// do not need the -Merge / finalizeAggregation guidance).
		err := errors.New(`clickhouse: unsupported column type "SimpleAggregateFunction(sum, UInt64)"`)
		require.Same(t, err, translateAggregateFunctionError(err))
	})

	t.Run("aggregate function error", func(t *testing.T) {
		err := errors.New(`read data: failed to decode block from 127.0.0.1:9000 (conn_id=26, compression=none): clickhouse: unsupported column type "AggregateFunction(argMin, Decimal(20, 6), DateTime64(3, 'UTC'))"`)
		got := translateAggregateFunctionError(err)
		require.Error(t, got)
		require.Contains(t, got.Error(), "cannot be decoded directly")
		require.Contains(t, got.Error(), "finalizeAggregation(col)")
		require.Contains(t, got.Error(), err.Error())
	})
}
