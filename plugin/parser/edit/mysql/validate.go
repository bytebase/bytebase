package mysql

import (
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
func (*SchemaEditor) ValidateDatabaseEdit(databaseEdit *api.DatabaseEdit) error {
	var addColumnContextList []*api.AddColumnContext
	var changeColumnContextList []*api.ChangeColumnContext

	for _, createTableContext := range databaseEdit.CreateTableList {
		addColumnContextList = append(addColumnContextList, createTableContext.AddColumnList...)
	}
	for _, alterTableContext := range databaseEdit.AlterTableList {
		addColumnContextList = append(addColumnContextList, alterTableContext.AddColumnList...)
		changeColumnContextList = append(changeColumnContextList, alterTableContext.ChangeColumnList...)
	}

	for _, addColumnContext := range addColumnContextList {
		columnType, err := transformColumnType(addColumnContext.Type)
		if err != nil {
			return errors.Errorf("invalid column type: %s", addColumnContext.Type)
		}
		if addColumnContext.Default != nil {
			if columnType.GetType() == mysql.TypeBlob || columnType.GetType() == mysql.TypeGeometry || columnType.GetType() == mysql.TypeJSON {
				return errors.Errorf("column type `%s` cannot have a default value", addColumnContext.Type)
			}
		}
	}

	for _, changeColumnContext := range changeColumnContextList {
		columnType, err := transformColumnType(changeColumnContext.Type)
		if err != nil {
			return errors.Errorf("invalid column type: %s", changeColumnContext.Type)
		}
		if changeColumnContext.Default != nil {
			if columnType.GetType() == mysql.TypeBlob || columnType.GetType() == mysql.TypeGeometry || columnType.GetType() == mysql.TypeJSON {
				return errors.Errorf("column type `%s` cannot have a default value", changeColumnContext.Type)
			}
		}
	}
	return nil
}
