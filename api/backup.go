package api

import (
	"context"
	"encoding/json"
)

// BackupStatus is the status of a backup.
type BackupStatus string

// BackupType is the type of a backup.
type BackupType string

// BackupStorageBackend is the storage backend of a backup.
type BackupStorageBackend string

const (
	// BackupStatusPendingCreate is the status for PENDING_CREATE.
	BackupStatusPendingCreate BackupStatus = "PENDING_CREATE"
	// BackupStatusDone is the status for DONE.
	BackupStatusDone BackupStatus = "DONE"
	// BackupTypeAutomatic is the type for automatic backup.
	BackupTypeAutomatic BackupType = "AUTOMATIC"
	// BackupTypeManual is the type for manual backup.
	BackupTypeManual BackupType = "MANUAL"
	// BackupStorageBackendLocal is the local storage backend for a backup.
	BackupStorageBackendLocal BackupStorageBackend = "LOCAL"
)

type Backup struct {
	ID int `jsonapi:"primary,table"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseId int
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Name           string `jsonapi:"attr,name"`
	Status         string `jsonapi:"attr,status"`
	Type           string `jsonapi:"attr,type"`
	StorageBackend string `jsonapi:"attr,storageBackend"`
	Path           string `jsonapi:"attr,path"`
	Comment        string `jsonapi:"attr,comment"`
}

type BackupCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name           string `jsonapi:"attr,name"`
	Status         string `jsonapi:"attr,status"`
	Type           string `jsonapi:"attr,type"`
	StorageBackend string `jsonapi:"attr,storageBackend"`
	Path           string `jsonapi:"attr,path"`
	Comment        string `jsonapi:"attr,comment"`
}

type BackupFind struct {
	ID *int

	// Related fields
	DatabaseId *int

	// Domain specific fields
	Name *string
}

func (find *BackupFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type BackupPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Status string
}

// BackupSetting is the backup setting for a database.
type BackupSetting struct {
	ID int `jsonapi:"primary,table"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseId int
	Database   *Database `jsonapi:"relation,database"`

	// Domain specific fields
	Enabled   int `jsonapi:"attr,enabled"`
	Hour      int `jsonapi:"attr,hour"`
	DayOfWeek int `jsonapi:"attr,dayOfWeek"`
}

// BackupSettingGet is the message to get a backup settings.
type BackupSettingGet struct {
	ID *int

	// Related fields
	DatabaseId *int

	// Domain specific fields
}

// BackupSettingSet is the message to set a backup settings.
type BackupSettingSet struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorId is the ID of the creator.
	CreatorId int

	// Related fields
	DatabaseId int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Enabled   int `jsonapi:"attr,enabled"`
	Hour      int `jsonapi:"attr,hour"`
	DayOfWeek int `jsonapi:"attr,dayOfWeek"`
}

// BackupSettingsMatch is the message to find backup settings matching the conditions.
type BackupSettingsMatch struct {
	Hour      int
	DayOfWeek int
}

// BackupService is the backend for backups.
type BackupService interface {
	CreateBackup(ctx context.Context, create *BackupCreate) (*Backup, error)
	FindBackup(ctx context.Context, find *BackupFind) (*Backup, error)
	FindBackupList(ctx context.Context, find *BackupFind) ([]*Backup, error)
	PatchBackup(ctx context.Context, patch *BackupPatch) (*Backup, error)
	GetBackupSetting(ctx context.Context, get *BackupSettingGet) (*BackupSetting, error)
	SetBackupSetting(ctx context.Context, setting *BackupSettingSet) (*BackupSetting, error)
	GetBackupSettingsMatch(ctx context.Context, match *BackupSettingsMatch) ([]*BackupSetting, error)
}
