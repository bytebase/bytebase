package databricks

import (
	"context"
	"io"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

const (
	schemaStmtFmt = "" +
		"--\n" +
		"-- %s structure for %s\n" +
		"--\n" +
		"%s;\n"
)

func (d *Driver) Dump(ctx context.Context, writer io.Writer, _ bool) (string, error) {
	catalogMap, err := d.listAllTables(ctx)
	if err != nil {
		return "", err
	}

	for catalogName, schemaMap := range catalogMap {
		for schemaName, tableMap := range schemaMap {
			for tblName, tblUnion := range tableMap {
				switch tblUnion.typeName {
				case catalog.TableTypeView:

				case catalog.TableTypeMaterializedView:

				case catalog.TableTypeExternal:

				case catalog.TableTypeManaged:

				default:
					// we do not sync streaming table.
					continue
				}

			}
		}

	}

	return "", nil
}
