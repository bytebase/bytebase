// Package export provides data export functionality for various formats (CSV, JSON, SQL, XLSX).
// It implements streaming export to minimize memory usage for large datasets.
package export

import (
	"bytes"
	"io"
	"time"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	timestampFormat   = "2006-01-02 15:04:05.000000"
	timestampTzFormat = time.RFC3339Nano
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

func formatTimestamp(ts *v1pb.RowValue_Timestamp) string {
	return ts.GoogleTimestamp.AsTime().Format(timestampFormat)
}

func formatTimestampTz(ts *v1pb.RowValue_TimestampTZ) string {
	t := ts.GoogleTimestamp.AsTime()
	z := time.FixedZone(ts.GetZone(), int(ts.GetOffset()))
	return t.In(z).Format(timestampTzFormat)
}
