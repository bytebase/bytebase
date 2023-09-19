package mysql

import (
	"fmt"

	"github.com/pingcap/tidb/parser/mysql"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
func (*SchemaEditor) ValidateDatabaseEdit(databaseEdit *api.DatabaseEdit) ([]*api.ValidateResult, error) {
	validateResultList := []*api.ValidateResult{}
	addColumnContextList := []*api.AddColumnContext{}
	changeColumnContextList := []*api.ChangeColumnContext{}

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
			validateResultList = append(validateResultList, &api.ValidateResult{
				Type:    api.ValidateErrorResult,
				Message: fmt.Sprintf("invalid column type `%s`", addColumnContext.Type),
			})
		}
		if columnType != nil && addColumnContext.Default != nil {
			// TEXT will be regarded as mysql.TypeBlob in the TiDB parser.
			if columnType.GetType() == mysql.TypeBlob || columnType.GetType() == mysql.TypeGeometry || columnType.GetType() == mysql.TypeJSON {
				validateResultList = append(validateResultList, &api.ValidateResult{
					Type:    api.ValidateErrorResult,
					Message: fmt.Sprintf("column type `%s` cannot have a default value", addColumnContext.Type),
				})
			}
		}
	}

	for _, changeColumnContext := range changeColumnContextList {
		columnType, err := transformColumnType(changeColumnContext.Type)
		if err != nil {
			validateResultList = append(validateResultList, &api.ValidateResult{
				Type:    api.ValidateErrorResult,
				Message: fmt.Sprintf("invalid column type `%s`", changeColumnContext.Type),
			})
		}
		if changeColumnContext.Default != nil {
			if columnType.GetType() == mysql.TypeBlob || columnType.GetType() == mysql.TypeGeometry || columnType.GetType() == mysql.TypeJSON {
				validateResultList = append(validateResultList, &api.ValidateResult{
					Type:    api.ValidateErrorResult,
					Message: fmt.Sprintf("column type `%s` cannot have a default value", changeColumnContext.Type),
				})
			}
		}
	}

	return validateResultList, nil
}
