package base

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type RestoreContext struct {
	InstanceID              string
	GetDatabaseMetadataFunc GetDatabaseMetadataFunc
	ListDatabaseNamesFunc   ListDatabaseNamesFunc
	IsCaseSensitive         bool
}

// ValidateBackupItem ensures backupItem and its SourceTable and TargetTable
// fields are non-nil.
func ValidateBackupItem(backupItem *storepb.PriorBackupDetail_Item) error {
	if backupItem == nil {
		return errors.New("backup item is nil")
	}
	if backupItem.SourceTable == nil {
		return errors.New("backup item source table is nil")
	}
	if backupItem.TargetTable == nil {
		return errors.New("backup item target table is nil")
	}
	return nil
}

func parseBackupDatabase(resourceName, role string) (string, error) {
	_, _, db, err := common.GetDatabaseResourceName(resourceName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %s database ID for %s", role, resourceName)
	}
	return db, nil
}

// ParseBackupSourceDatabase extracts the database ID from backupItem.SourceTable.Database.
func ParseBackupSourceDatabase(backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	return parseBackupDatabase(backupItem.SourceTable.Database, "source")
}

// ParseBackupTargetDatabase extracts the database ID from backupItem.TargetTable.Database.
func ParseBackupTargetDatabase(backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	return parseBackupDatabase(backupItem.TargetTable.Database, "target")
}

// ParseBackupItemDatabases extracts the source and target database IDs from backupItem.
func ParseBackupItemDatabases(backupItem *storepb.PriorBackupDetail_Item) (sourceDatabase, targetDatabase string, err error) {
	sourceDatabase, err = ParseBackupSourceDatabase(backupItem)
	if err != nil {
		return "", "", err
	}
	targetDatabase, err = ParseBackupTargetDatabase(backupItem)
	if err != nil {
		return "", "", err
	}
	return sourceDatabase, targetDatabase, nil
}
