package databricks

import (
	"context"
	"io"
)

const (
	schemaStmtFmt = "" +
		"--\n" +
		"-- %s structure for %s\n" +
		"--\n" +
		"%s;\n"
)

func (d *Driver) Dump(ctx context.Context, writer io.Writer, _ bool) (string, error) {
	// dump tables.
	// catalogMap, err := d.listTables(ctx)
	// if err != nil {
	// 	return "", err
	// }

	// for catalogName, _ := range catalogMap {

	// }

	if _, err := io.WriteString(writer, ""); err != nil {
		return "", err
	}

	return "", nil
}
