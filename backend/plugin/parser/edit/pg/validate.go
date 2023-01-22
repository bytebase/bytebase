package pg

import (
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
func (*SchemaEditor) ValidateDatabaseEdit(_ *api.DatabaseEdit) ([]*api.ValidateResult, error) {
	// Because Postgres has a custom data type, so skip directly here.
	return []*api.ValidateResult{}, nil
}
