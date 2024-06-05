package databricks

import (
	"context"
	"io"
)

// Dump dumps the database.
func (*DatabricksDriver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}
