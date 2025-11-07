// Package export provides data export functionality for various formats (CSV, JSON, SQL, XLSX).
// It implements streaming export to minimize memory usage for large datasets.
package export

import (
	"bytes"
	"io"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Writer is a function type that writes query results to a writer.
type Writer func(w io.Writer, result *v1pb.QueryResult) error

// exportToBytes is a helper function that exports to a byte slice using a writer function.
func exportToBytes(result *v1pb.QueryResult, writerFunc Writer) ([]byte, error) {
	var buf bytes.Buffer
	if err := writerFunc(&buf, result); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
