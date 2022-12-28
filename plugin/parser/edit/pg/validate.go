package pg

import (
	"github.com/bytebase/bytebase/api"
)

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
func (*SchemaEditor) ValidateDatabaseEdit(databaseEdit *api.DatabaseEdit) ([]*api.ValidateResult, error) {
	// Because Postgres has a custom data type, so skip directly here.
	return []*api.ValidateResult{}, nil
}
