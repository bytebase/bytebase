//go:build release
// +build release

package cmd

import (
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
)

func activeProfile(dataDir string, backupMeta backupMeta) server.Profile {
	p := getBaseProfile()
	p.Mode = common.ReleaseModeProd
	p.PgUser = "bb"
	p.DataDir = dataDir
	p.BackupRunnerInterval = 10 * time.Minute
	p.MetricConnectionKey = "so9lLwj5zLjH09sxNabsyVNYSsAHn68F"

	p.BackupStorageBackend = backupMeta.storageBackend
	p.BackupRegion = backupMeta.region
	p.BackupBucket = backupMeta.bucket
	p.BackupCredentialFile = backupMeta.credentialFile
	return p
}
