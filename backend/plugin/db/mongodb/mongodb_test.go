package mongodb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetMongoDBConnectionURL(t *testing.T) {
	tests := []struct {
		connConfig db.ConnectionConfig
		want       string
	}{
		{
			connConfig: db.ConnectionConfig{
				Host:     "localhost",
				Port:     "27017",
				Username: "",
				Password: "",
			},
			want: "mongodb://localhost:27017/?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:     "localhost",
				Port:     "27017",
				Username: "",
				Password: "",
				Database: "sampleDB",
			},
			want: "mongodb://localhost:27017/sampleDB?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:     "cluster0.sample.mongodb.net",
				Username: "bytebase",
				Password: "passwd",
				Database: "sampleDB",
				SRV:      true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "cluster0.sample.mongodb.net",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "",
				SRV:                    true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "cluster0.sample.mongodb.net",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "admin",
				SRV:                    true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?authSource=admin",
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		got := getMongoDBConnectionURI(tt.connConfig)
		a.Equal(tt.want, got)
	}
}

func TestIsMongoStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      bool
	}{
		{
			statement: `show collections`,
			want:      false,
		},
		{
			statement: `db.cpl_station_info.find().limit(100)`,
			want:      true,
		},
		{
			statement: ` 
			  db.cpl_station_info.find().limit(100);
			`,
			want: true,
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		got := isMongoStatement(tt.statement)
		a.Equal(tt.want, got, tt.statement)
	}
}

func TestGetSimpleStatementResult(t *testing.T) {
	v1 := `{
	"age": 13,
	"groups": [
		"basketball",
		"swimming"
	],
	"name": "danny",
	"tree": {
		"a": "a",
		"b": 1
	}
}`
	v2 := `{
	"flower": 123
}`

	tests := []struct {
		data string
		want *v1pb.QueryResult
	}{
		{
			data: `{"_id":{"$oid":"64c0b8c4e65c51195e0584b2"},"name":"danny","age":13,"groups":["basketball","swimming"],"tree":{"a":"a","b":1}}`,
			want: &v1pb.QueryResult{
				ColumnNames:     []string{"_id", "result"},
				ColumnTypeNames: []string{"TEXT", "TEXT"},
				Rows: []*v1pb.QueryRow{{
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "64c0b8c4e65c51195e0584b2"}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: v1}},
					},
				}},
			},
		},
		{
			data: `[{"_id":{"$oid":"64c0b8c4e65c51195e0584b2"},"name":"danny","age":13,"groups":["basketball","swimming"],"tree":{"a":"a","b":1}},{"_id":{"$oid":"64c1de7e85c563e625f217d5"},"flower":123}]`,
			want: &v1pb.QueryResult{
				ColumnNames:     []string{"_id", "result"},
				ColumnTypeNames: []string{"TEXT", "TEXT"},
				Rows: []*v1pb.QueryRow{{
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "64c0b8c4e65c51195e0584b2"}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: v1}},
					},
				}, {
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "64c1de7e85c563e625f217d5"}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: v2}},
					},
				}},
			},
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		result, err := getSimpleStatementResult([]byte(tt.data))
		a.NoError(err)
		diff := cmp.Diff(tt.want, result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
		a.Equal("", diff)
	}
}
