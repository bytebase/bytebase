package mssql

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_MSSQL, GetDatabaseMetadata)
}

type tableKey struct {
	schema string
	table  string
}
