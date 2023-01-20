package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/api"
)

func TestValidateDatabaseEditColumnType(t *testing.T) {
	tests := []struct {
		databaseEdit       *api.DatabaseEdit
		validateResultList []*api.ValidateResult
	}{
		{
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t1",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
				},
			},
			validateResultList: []*api.ValidateResult{},
		},
		{
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t1",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int123",
							},
						},
					},
				},
			},
			validateResultList: []*api.ValidateResult{
				{
					Type:    api.ValidateErrorResult,
					Message: "invalid column type `int123`",
				},
			},
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		validateResultList, err := mysqlEditor.ValidateDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.validateResultList, validateResultList)
	}
}

func TestValidateDatabaseEditColumnTypeAndDefault(t *testing.T) {
	defaultValue := "default_value"
	tests := []struct {
		databaseEdit       *api.DatabaseEdit
		validateResultList []*api.ValidateResult
	}{
		{
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t1",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "TEXT",
							},
						},
					},
				},
			},
			validateResultList: []*api.ValidateResult{},
		},
		{
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t1",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:    "id",
								Type:    "TEXT",
								Default: &defaultValue,
							},
						},
					},
				},
			},
			validateResultList: []*api.ValidateResult{
				{
					Type:    api.ValidateErrorResult,
					Message: "column type `TEXT` cannot have a default value",
				},
			},
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		validateResultList, err := mysqlEditor.ValidateDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.validateResultList, validateResultList)
	}
}
