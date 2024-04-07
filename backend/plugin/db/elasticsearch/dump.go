package elasticsearch

import (
	"context"
	"io"
)

// Dump() is not applicable to Elasticsearch.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}

// Restore() is not applicable to Elasticsearch.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return nil
}
