package mssql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

type fakeRowEstimatorDriver struct {
	gotStatement string
	rows         int64
	err          error
}

func (*fakeRowEstimatorDriver) Ping(context.Context) error { return nil }
func (*fakeRowEstimatorDriver) Close() error               { return nil }

func (f *fakeRowEstimatorDriver) CountAffectedRows(_ context.Context, statement string) (int64, error) {
	f.gotStatement = statement
	return f.rows, f.err
}

// TestCountAffectedRows pins the adapter's forwarding seam to omni's
// RowEstimator, including feeding the T-SQL changed-resource extractor's
// emitted DML sample through without mutation.
func TestCountAffectedRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("forwards statement and count", func(t *testing.T) {
		t.Parallel()
		fake := &fakeRowEstimatorDriver{rows: 300}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		const stmt = "DELETE FROM dbo.big WHERE s_id <= 3;"
		rows, err := driver.CountAffectedRows(ctx, stmt)
		require.NoError(t, err)
		require.Equal(t, int64(300), rows)
		require.Equal(t, stmt, fake.gotStatement)
	})

	t.Run("propagates estimator error", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("Invalid object name 'dbo.missing'")
		fake := &fakeRowEstimatorDriver{err: wantErr}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		_, err := driver.CountAffectedRows(ctx, "DELETE FROM dbo.missing;")
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("forwards extractor DML sample verbatim", func(t *testing.T) {
		t.Parallel()
		const statement = "-- header comment\nUPDATE dbo.t SET c1 = c1 + 1 WHERE id > 2;"

		stmts, err := parserbase.ParseStatements(storepb.Engine_MSSQL, statement)
		require.NoError(t, err)
		summary, err := parserbase.ExtractChangedResources(
			storepb.Engine_MSSQL, "resource_change_test", "dbo", nil, parserbase.ExtractASTs(stmts), statement)
		require.NoError(t, err)
		require.Len(t, summary.DMLStatements, 1)

		fake := &fakeRowEstimatorDriver{rows: 3}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		rows, err := driver.CountAffectedRows(ctx, summary.DMLStatements[0])
		require.NoError(t, err)
		require.Equal(t, int64(3), rows)
		require.Equal(t, summary.DMLStatements[0], fake.gotStatement)
	})
}
