package api

import (
	"context"
	"encoding/json"
)

// BackupStatus is the status of a backup.
type BackupStatus string

const (
	// BackupStatusPendingCreate is the status for PENDING_CREATE.
	BackupStatusPendingCreate BackupStatus = "PENDING_CREATE"
	// BackupStatusDone is the status for DONE.
	BackupStatusDone BackupStatus = "DONE"
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

type BackupService interface {
	CreateBackup(ctx context.Context, create *BackupCreate) (*Backup, error)
	FindBackup(ctx context.Context, find *BackupFind) (*Backup, error)
	FindBackupList(ctx context.Context, find *BackupFind) ([]*Backup, error)
	PatchBackup(ctx context.Context, patch *BackupPatch) (*Backup, error)
}
