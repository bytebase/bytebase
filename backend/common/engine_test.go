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
