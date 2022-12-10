package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/api"
)

func TestValidateDatabaseEditColumnType(t *testing.T) {
	tests := []struct {
		databaseEdit *api.DatabaseEdit
		errorMessage string
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
			errorMessage: "",
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
			errorMessage: "invalid column type `int123`",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		err := mysqlEditor.ValidateDatabaseEdit(test.databaseEdit)
		if err != nil {
			assert.Equal(t, test.errorMessage, err.Error())
		} else {
			assert.Equal(t, test.errorMessage, "")
		}
	}
}

func TestValidateDatabaseEditColumnTypeAndDefault(t *testing.T) {
	defaultValue := "123"
	tests := []struct {
		databaseEdit *api.DatabaseEdit
		errorMessage string
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
			errorMessage: "",
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
								Default: nil,
							},
						},
					},
				},
			},
			errorMessage: "",
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
			errorMessage: "column type `TEXT` cannot have a default value",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		err := mysqlEditor.ValidateDatabaseEdit(test.databaseEdit)
		if err != nil {
			assert.Equal(t, test.errorMessage, err.Error())
		} else {
			assert.Equal(t, test.errorMessage, "")
		}
	}
}
