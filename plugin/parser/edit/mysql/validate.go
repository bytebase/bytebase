package mysql

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
func (*SchemaEditor) ValidateDatabaseEdit(databaseEdit *api.DatabaseEdit) error {
	for _, createTableContext := range databaseEdit.CreateTableList {
		for _, addColumnContext := range createTableContext.AddColumnList {
			if _, err := transformColumnType(addColumnContext.Type); err != nil {
				return errors.Errorf("invalid column type: %s", addColumnContext.Type)
			}
		}
	}
	for _, alterTableContext := range databaseEdit.AlterTableList {
		for _, addColumnContext := range alterTableContext.AddColumnList {
			if _, err := transformColumnType(addColumnContext.Type); err != nil {
				return errors.Errorf("invalid column type: %s", addColumnContext.Type)
			}
		}
		for _, changeColumnContext := range alterTableContext.ChangeColumnList {
			if _, err := transformColumnType(changeColumnContext.Type); err != nil {
				return errors.Errorf("invalid column type: %s", changeColumnContext.Type)
			}
		}
	}
	return nil
}
