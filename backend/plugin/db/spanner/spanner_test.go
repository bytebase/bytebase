package spanner

import (
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/stretchr/testify/require"
)

func TestGetDatabaseFromDSN(t *testing.T) {
	tests := []struct {
		dsn   string
		match bool
		want  string
	}{
		{
			dsn:   "projects/p/instances/i/databases/d",
			match: true,
			want:  "d",
		},
		{
			dsn:   "projects/p/instances/i/databases/",
			match: false,
			want:  "",
		},
	}
	a := require.New(t)
	for i := range tests {
		test := tests[i]
		got, err := getDatabaseFromDSN(test.dsn)
		if test.match {
			a.NoError(err)
			a.Equal(test.want, got)
		} else {
			a.Error(err)
		}
	}
}

func TestGetColumnTypeName(t *testing.T) {
	tests := []struct {
		spannerType spannerpb.Type
		want        string
	}{
		{
			spannerType: spannerpb.Type{
				Code: spannerpb.TypeCode_JSON,
			},
			want: "JSON",
		},
		{
			spannerType: spannerpb.Type{
				Code: spannerpb.TypeCode_DATE,
			},
			want: "DATE",
		},
		{
			spannerType: spannerpb.Type{
				Code: spannerpb.TypeCode_TIMESTAMP,
			},
			want: "TIMESTAMP",
		},
		{
			spannerType: spannerpb.Type{
				Code: spannerpb.TypeCode_ARRAY,
				ArrayElementType: &spannerpb.Type{
					Code: spannerpb.TypeCode_BYTES,
				},
			},
			want: "[]BYTES",
		},
	}
	a := require.New(t)
	for i := range tests {
		got, err := getColumnTypeName(&tests[i].spannerType)
		a.NoError(err)
		a.Equal(tests[i].want, got)
	}
}
