package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func TestValidateDatabaseEditColumnType(t *testing.T) {
	tests := []struct {
		databaseEdit       *api.DatabaseEdit
		validateResultList []*api.ValidateResult
	}{}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		validateResultList, err := postgresEditor.ValidateDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.validateResultList, validateResultList)
	}
}
