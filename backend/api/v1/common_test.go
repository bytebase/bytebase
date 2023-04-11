package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidResourceID(t *testing.T) {
	tests := []struct {
		resourceID string
		want       bool
	}{
		{
			resourceID: "hello123",
			want:       true,
		},
		{
			resourceID: "hello-123",
			want:       true,
		},
		{
			resourceID: "你好",
			want:       false,
		},
		{
			resourceID: "123abc",
			want:       false,
		},
		{
			resourceID: "a1234567890123456789012345678901234567890123456789012345678901234567890",
			want:       false,
		},
	}

	for _, test := range tests {
		got := isValidResourceID(test.resourceID)
		require.Equal(t, test.want, got, test.resourceID)
	}
}

func TestGetFilter(t *testing.T) {
	tests := []struct {
		filter    string
		filterKey string
		want      string
		wantErr   bool
	}{
		{
			filter:    "",
			filterKey: "",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= "projects/abc".`,
			filterKey: "project",
			want:      "projects/abc",
			wantErr:   false,
		},
		{
			filter:    `project= "projects/abc".`,
			filterKey: "instance",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= abc.`,
			filterKey: "project",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= abc`,
			filterKey: "project",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= "projects/abc"`,
			filterKey: "project",
			want:      "",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		value, err := getFilter(test.filter, test.filterKey)
		if test.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.want, value)
		}
	}
}

func TestGetEnvironmentInstanceDatabaseID(t *testing.T) {
	environmentID, instanceID, err := getEnvironmentInstanceID("environments/e1/instances/i2")
	require.NoError(t, err)
	require.Equal(t, "e1", environmentID)
	require.Equal(t, "i2", instanceID)

	_, _, err = getEnvironmentInstanceID("environments/e1/instances/i2/databases/d3")
	require.Error(t, err)
}

func TestGetRiskID(t *testing.T) {
	tests := []struct {
		name string
		want int64
	}{
		{
			name: "risks/1234",
			want: 1234,
		},
		{
			name: "risks/12345678901",
			want: 12345678901,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		got, err := getRiskID(test.name)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestGetEBNFTokens(t *testing.T) {
	testCases := []struct {
		input   string
		key     string
		want    []string
		wantErr bool
	}{
		{
			input: `resource="environments/e1/instances/i2".`,
			key:   "resource",
			want: []string{
				"environments/e1/instances/i2",
			},
			wantErr: false,
		},
		{
			input: `resource="environments/e1/instances/i2/databases/db3".`,
			key:   "resource",
			want: []string{
				"environments/e1/instances/i2/databases/db3",
			},
			wantErr: false,
		},
		{
			input: `type="DATABASE_BACKUP_MISSING" | "DATABASE_BACKUP_FAILED".`,
			key:   "type",
			want: []string{
				"DATABASE_BACKUP_MISSING",
				"DATABASE_BACKUP_FAILED",
			},
			wantErr: false,
		},
		{
			input: `a="1" | "2". b="3" | "4".`,
			key:   "b",
			want: []string{
				"3",
				"4",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		got, err := getEBNFTokens(tc.input, tc.key)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		}
	}
}

func TestParseFilter(t *testing.T) {
	testCases := []struct {
		input string
		want  []expression
		err   error
	}{
		{
			input: `resource="environments/e1/instances/i2"`,
			want: []expression{
				{
					key:        "resource",
					comparator: comparatorTypeEqual,
					value:      "environments/e1/instances/i2",
				},
			},
		},
		{
			input: `project = "p1" && start_time>="2020-01-01T00:00:00Z" && start_time<2020-01-02T00:00:00Z`,
			want: []expression{
				{
					key:        "project",
					comparator: comparatorTypeEqual,
					value:      "p1",
				},
				{
					key:        "start_time",
					comparator: comparatorTypeGreaterEqual,
					value:      "2020-01-01T00:00:00Z",
				},
				{
					key:        "start_time",
					comparator: comparatorTypeLess,
					value:      "2020-01-02T00:00:00Z",
				},
			},
		},
	}

	for _, test := range testCases {
		got, err := parseFilter(test.input)
		if test.err != nil {
			require.EqualError(t, err, test.err.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		}
	}
}

func TestParseOrderBy(t *testing.T) {
	testCases := []struct {
		input string
		want  []orderByKey
		err   error
	}{
		{
			input: "start_time",
			want: []orderByKey{
				{
					key:      "start_time",
					isAscend: true,
				},
			},
		},
		{
			input: "start_time desc",
			want: []orderByKey{
				{
					key:      "start_time",
					isAscend: false,
				},
			},
		},
		{
			input: "start_time desc, count",
			want: []orderByKey{
				{
					key:      "start_time",
					isAscend: false,
				},
				{
					key:      "count",
					isAscend: true,
				},
			},
		},
	}

	for _, test := range testCases {
		got, err := parseOrderBy(test.input)
		if test.err != nil {
			require.EqualError(t, err, test.err.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		}
	}
}
