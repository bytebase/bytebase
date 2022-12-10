package mysql

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
// TODO(steven): return a validation result struct list instead of string.
func (*SchemaEditor) ValidateDatabaseEdit(databaseEdit *api.DatabaseEdit) error {
	var invalidMessageList []string
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
			invalidMessageList = append(invalidMessageList, fmt.Sprintf("invalid column type `%s`", addColumnContext.Type))
		}
		if addColumnContext.Default != nil {
			// TEXT will be regarded as mysql.TypeBlob in the TiDB parser.
			if columnType.GetType() == mysql.TypeBlob || columnType.GetType() == mysql.TypeGeometry || columnType.GetType() == mysql.TypeJSON {
				invalidMessageList = append(invalidMessageList, fmt.Sprintf("column type `%s` cannot have a default value", addColumnContext.Type))
			}
		}
	}

	for _, changeColumnContext := range changeColumnContextList {
		columnType, err := transformColumnType(changeColumnContext.Type)
		if err != nil {
			invalidMessageList = append(invalidMessageList, fmt.Sprintf("invalid column type `%s`", changeColumnContext.Type))
		}
		if changeColumnContext.Default != nil {
			if columnType.GetType() == mysql.TypeBlob || columnType.GetType() == mysql.TypeGeometry || columnType.GetType() == mysql.TypeJSON {
				invalidMessageList = append(invalidMessageList, fmt.Sprintf("column type `%s` cannot have a default value", changeColumnContext.Type))
			}
		}
	}

	if len(invalidMessageList) != 0 {
		return errors.New(strings.Join(invalidMessageList, "\n"))
	}
	return nil
}
