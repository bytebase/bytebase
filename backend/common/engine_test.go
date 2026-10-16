//nolint:revive
package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestEngineSupportPriorBackupMariaDB(t *testing.T) {
	require.True(t, EngineSupportPriorBackup(storepb.Engine_MARIADB))
}

func TestBackupDatabaseNameOfEngineMariaDB(t *testing.T) {
	require.Equal(t, "bbdataarchive", BackupDatabaseNameOfEngine(storepb.Engine_MARIADB))
}

func TestEngineSupportSDLExport(t *testing.T) {
	tests := []struct {
		engine storepb.Engine
		want   bool
	}{
		{storepb.Engine_MYSQL, true},
		{storepb.Engine_POSTGRES, true},
		{storepb.Engine_COCKROACHDB, true},
		// OceanBase shares MySQL's single-file SDL branch but is deliberately excluded
		// from the SDL paths until validated.
		{storepb.Engine_OCEANBASE, false},
		{storepb.Engine_TIDB, false},
		{storepb.Engine_ORACLE, false},
		{storepb.Engine_CLICKHOUSE, false},
	}
	for _, tt := range tests {
		t.Run(tt.engine.String(), func(t *testing.T) {
			require.Equal(t, tt.want, EngineSupportSDLExport(tt.engine))
		})
	}
}

func TestEngineSupportSQLReview(t *testing.T) {
	tests := []struct {
		engine storepb.Engine
		want   bool
	}{
		{storepb.Engine_POSTGRES, true},
		{storepb.Engine_MYSQL, true},
		{storepb.Engine_BIGQUERY, true},
		{storepb.Engine_SPANNER, false},
		{storepb.Engine_MONGODB, false},
	}
	for _, tt := range tests {
		t.Run(tt.engine.String(), func(t *testing.T) {
			require.Equal(t, tt.want, EngineSupportSQLReview(tt.engine))
		})
	}
}

// The plan-check executor reports "Statement advise is not supported" and skips
// SQL review entirely when this gate is false, so every engine with advisor
// rules must be listed here.
func TestEngineSupportStatementAdvise(t *testing.T) {
	tests := []struct {
		engine storepb.Engine
		want   bool
	}{
		{storepb.Engine_POSTGRES, true},
		{storepb.Engine_MYSQL, true},
		{storepb.Engine_BIGQUERY, true},
		{storepb.Engine_SPANNER, false},
		{storepb.Engine_MONGODB, false},
	}
	for _, tt := range tests {
		t.Run(tt.engine.String(), func(t *testing.T) {
			require.Equal(t, tt.want, EngineSupportStatementAdvise(tt.engine))
		})
	}
}
