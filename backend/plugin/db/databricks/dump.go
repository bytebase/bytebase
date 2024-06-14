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

func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}
