//go:build !release
// +build !release

package cmd

import (
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string, backupMeta backupMeta) server.Profile {
	p := getBaseProfile()
	p.Mode = common.ReleaseModeDev
	p.PgUser = "bbdev"
	p.DataDir = dataDir
	p.BackupRunnerInterval = 10 * time.Second
	p.MetricConnectionKey = "3zcZLeX3ahvlueEJqNyJysGfVAErsjjT"

	p.BackupStorageBackend = backupMeta.storageBackend
	p.BackupRegion = backupMeta.region
	p.BackupBucket = backupMeta.bucket
	p.CredentialsFile = backupMeta.credentialsFile
	return p
}
