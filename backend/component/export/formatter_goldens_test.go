package export

import (
	"bufio"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// goldensFormatterDir locates the shared TSV fixtures both Go and TS read from.
func goldensFormatterDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "frontend", "src", "utils", "sql-download", "__tests__", "goldens", "formatters")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	t.Fatal("could not locate frontend/src/utils/sql-download/__tests__/goldens/formatters")
	return ""
}

func readTSVLines(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	var out [][]string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.Split(line, "\t"))
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return out
}

func TestFloat32FormatterGolden(t *testing.T) {
	t.Parallel()
	for _, row := range readTSVLines(t, filepath.Join(goldensFormatterDir(t), "float32.tsv")) {
		if len(row) != 2 {
			t.Fatalf("float32.tsv malformed row: %v", row)
		}
		bits, err := hex.DecodeString(row[0])
		if err != nil || len(bits) != 4 {
			t.Fatalf("invalid hex %q: %v", row[0], err)
		}
		u := uint32(bits[0])<<24 | uint32(bits[1])<<16 | uint32(bits[2])<<8 | uint32(bits[3])
		f := math.Float32frombits(u)
		got := strconv.FormatFloat(float64(f), 'f', -1, 32)
		if got != row[1] {
			t.Errorf("bits=%s: got %q, want %q", row[0], got, row[1])
		}
	}
}

func TestTimestampFormatterGolden(t *testing.T) {
	t.Parallel()
	for _, row := range readTSVLines(t, filepath.Join(goldensFormatterDir(t), "timestamp.tsv")) {
		if len(row) != 3 {
			t.Fatalf("timestamp.tsv malformed row: %v", row)
		}
		sec, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			t.Fatalf("parse seconds %q: %v", row[0], err)
		}
		nanos, err := strconv.ParseInt(row[1], 10, 32)
		if err != nil {
			t.Fatalf("parse nanos %q: %v", row[1], err)
		}
		ts := &v1pb.RowValue_Timestamp{
			GoogleTimestamp: &timestamppb.Timestamp{Seconds: sec, Nanos: int32(nanos)},
		}
		got := formatTimestamp(ts)
		if got != row[2] {
			t.Errorf("seconds=%d nanos=%d: got %q, want %q", sec, nanos, got, row[2])
		}
	}
}

func TestTimestampTZFormatterGolden(t *testing.T) {
	t.Parallel()
	for _, row := range readTSVLines(t, filepath.Join(goldensFormatterDir(t), "timestamptz.tsv")) {
		if len(row) != 5 {
			t.Fatalf("timestamptz.tsv malformed row: %v", row)
		}
		sec, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			t.Fatalf("parse seconds %q: %v", row[0], err)
		}
		nanos, err := strconv.ParseInt(row[1], 10, 32)
		if err != nil {
			t.Fatalf("parse nanos %q: %v", row[1], err)
		}
		zone := row[2]
		if zone == "-" {
			zone = ""
		}
		offset, err := strconv.ParseInt(row[3], 10, 32)
		if err != nil {
			t.Fatalf("parse offset %q: %v", row[3], err)
		}
		ts := &v1pb.RowValue_TimestampTZ{
			GoogleTimestamp: &timestamppb.Timestamp{Seconds: sec, Nanos: int32(nanos)},
			Zone:            zone,
			Offset:          int32(offset),
		}
		got := formatTimestampTz(ts)
		if got != row[4] {
			t.Errorf("seconds=%d nanos=%d zone=%q offset=%d: got %q, want %q", sec, nanos, zone, offset, got, row[4])
		}
	}
}
