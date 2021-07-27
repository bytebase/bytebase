package api

import (
	"context"
	"encoding/json"
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

type BackupService interface {
	FindBackupList(ctx context.Context, find *BackupFind) ([]*Backup, error)
}
