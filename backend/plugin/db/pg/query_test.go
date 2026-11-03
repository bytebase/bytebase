package pg

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestBuildTimestamptzRowValue(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		input      time.Time
		wantString string // non-empty means we expect a string fallback with this value
	}{
		{
			name:     "valid timestamp",
			typeName: "TIMESTAMP",
			input:    time.Date(2026, 5, 11, 12, 34, 56, 0, time.UTC),
		},
		{
			name:     "valid timestamptz",
			typeName: "TIMESTAMPTZ",
			input:    time.Date(2026, 5, 11, 12, 34, 56, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name:       "BC timestamp falls back to PG-formatted string",
			typeName:   "TIMESTAMP",
			input:      time.Date(0, 12, 31, 15, 0, 0, 0, time.UTC), // Go year 0 == 1 BC
			wantString: "0001-12-31 15:00:00 BC",
		},
		{
			name:       "year 10000 timestamp omits zone",
			typeName:   "TIMESTAMP",
			input:      time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
			wantString: "10000-01-01 00:00:00",
		},
		{
			name:       "year 10000 timestamptz keeps offset",
			typeName:   "TIMESTAMPTZ",
			input:      time.Date(10000, 5, 11, 12, 34, 56, 0, time.FixedZone("", 2*60*60)),
			wantString: "10000-05-11 12:34:56+02:00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTimestamptzRowValue(tc.typeName, tc.input, 6)

			// The whole point of the fix: marshaling must never fail.
			_, err := protojson.Marshal(got)
			require.NoError(t, err)

			if tc.wantString != "" {
				require.Equal(t, tc.wantString, got.GetStringValue())
				return
			}
			if tc.typeName == "TIMESTAMP" {
				require.NotNil(t, got.GetTimestampValue())
			} else {
				require.NotNil(t, got.GetTimestampTzValue())
			}
		})
	}
}

func TestPadZeroes(t *testing.T) {
	tests := []struct {
		name     string
		rawStr   string
		acc      int
		expected string
	}{
		{
			name:     "No decimal part",
			rawStr:   "12:34:56",
			acc:      6,
			expected: "12:34:56",
		},
		{
			name:     "Already correct precision",
			rawStr:   "12:34:56.123456",
			acc:      6,
			expected: "12:34:56.123456",
		},
		{
			name:     "Less precision",
			rawStr:   "12:34:56.123",
			acc:      6,
			expected: "12:34:56.123000",
		},
		{
			name:     "More precision",
			rawStr:   "12:34:56.123456789",
			acc:      6,
			expected: "12:34:56.123456789",
		},
		{
			name:     "With timezone",
			rawStr:   "12:34:56.123+02:00",
			acc:      6,
			expected: "12:34:56.123000+02:00",
		},
		{
			name:     "With negative timezone",
			rawStr:   "12:34:56.123-02:00",
			acc:      6,
			expected: "12:34:56.123000-02:00",
		},
		{
			name:     "Invalid format: timezone before decimal (edge case fix)",
			rawStr:   "12:34-05:56.123",
			acc:      6,
			expected: "12:34-05:56.123000",
		},
		{
			name:     "Negative interval: leading minus should be ignored",
			rawStr:   "-00:04:37.530865",
			acc:      6,
			expected: "-00:04:37.530865",
		},
		{
			name:     "Negative interval with short precision",
			rawStr:   "-00:02:45.25",
			acc:      6,
			expected: "-00:02:45.250000",
		},
		{
			name:     "Negative interval without decimal",
			rawStr:   "-00:05:00",
			acc:      6,
			expected: "-00:05:00",
		},
		{
			name:     "Negative interval with timezone (minus after decimal)",
			rawStr:   "-12:34:56.123-05:00",
			acc:      6,
			expected: "-12:34:56.123-05:00",
		},
	}

	for _, test := range tests {
		got := padZeroes(test.rawStr, test.acc)
		require.Equal(t, test.expected, got)
	}
}

// TestConvertValueTimestampFromDriver reads timestamps through the driver rather than
// from a constructed time.Time, exercising the session time zone the way QueryConn does:
// read once per statement, then applied to every timestamptz in the result. It also
// covers ColumnType.DecimalSize reporting the column's fractional-second scale, which
// Bytebase's pgx fork adds and upstream does not, so a pgx upgrade that loses that patch
// fails here instead of silently changing query results.
func TestConvertValueTimestampFromDriver(t *testing.T) {
	const (
		sessionZone   = "Asia/Shanghai"
		sessionOffset = 8 * 60 * 60
	)

	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)

	// SET applies to one session, so hold a single connection for the query.
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()
	_, err = conn.ExecContext(ctx, "SET TimeZone TO '"+sessionZone+"'")
	require.NoError(t, err)

	// QueryConn reads the zone once per statement, after any preceding SET.
	location := util.PgSessionLocation(conn)
	require.Equal(t, sessionZone, location.String())

	rows, err := conn.QueryContext(ctx, `SELECT
		'2026-03-04 05:06:07.891234+00'::timestamptz    AS tz_default,
		'2026-03-04 05:06:07.891234+00'::timestamptz(3) AS tz_scale3,
		'2026-03-04 05:06:07.891234'::timestamp         AS ts_default`)
	require.NoError(t, err)
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	require.NoError(t, err)
	require.Len(t, columnTypes, 3)

	require.True(t, rows.Next())
	dest := make([]any, len(columnTypes))
	for i, columnType := range columnTypes {
		dest[i] = makeValueByTypeName(columnType.DatabaseTypeName(), columnType)
	}
	require.NoError(t, rows.Scan(dest...))
	require.NoError(t, rows.Err())

	values := make(map[string]*v1pb.RowValue, len(columnTypes))
	for i, columnType := range columnTypes {
		values[columnType.Name()] = convertValue(columnType.DatabaseTypeName(), columnType, dest[i], location)
	}

	tzDefault := values["tz_default"].GetTimestampTzValue()
	require.NotNil(t, tzDefault)
	require.Equal(t, "CST", tzDefault.Zone)
	require.EqualValues(t, sessionOffset, tzDefault.Offset)
	require.EqualValues(t, 6, tzDefault.Accuracy)
	require.Equal(t, "2026-03-04T05:06:07.891234Z", tzDefault.GoogleTimestamp.AsTime().Format(time.RFC3339Nano))

	// Postgres rounds the fixture's .891234 to .891 for timestamptz(3).
	tzScale3 := values["tz_scale3"].GetTimestampTzValue()
	require.NotNil(t, tzScale3)
	require.Equal(t, "CST", tzScale3.Zone)
	require.EqualValues(t, sessionOffset, tzScale3.Offset)
	require.EqualValues(t, 3, tzScale3.Accuracy)
	require.Equal(t, "2026-03-04T05:06:07.891Z", tzScale3.GoogleTimestamp.AsTime().Format(time.RFC3339Nano))

	// timestamp without time zone carries no zone, so it is not shifted.
	tsDefault := values["ts_default"].GetTimestampValue()
	require.NotNil(t, tsDefault)
	require.EqualValues(t, 6, tsDefault.Accuracy)
	require.Equal(t, "2026-03-04T05:06:07.891234Z", tsDefault.GoogleTimestamp.AsTime().Format(time.RFC3339Nano))
}
