package databricks

import (
	"context"
	"io"
)

func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}
