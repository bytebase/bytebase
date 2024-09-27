package mongodb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetMongoDBConnectionURL(t *testing.T) {
	tests := []struct {
		connConfig db.ConnectionConfig
		want       string
	}{
		{
			connConfig: db.ConnectionConfig{
				Host:                 "localhost",
				Port:                 "27017",
				Username:             "",
				Password:             "",
				MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb://localhost:27017/?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                 "localhost",
				Port:                 "27017",
				Username:             "",
				Password:             "",
				DirectConnection:     true,
				MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb://localhost:27017/?appName=bytebase&authSource=admin&directConnection=true",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                 "localhost",
				Port:                 "27017",
				Username:             "",
				Password:             "",
				Database:             "sampleDB",
				MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb://localhost:27017/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                 "cluster0.sample.mongodb.net",
				Username:             "bytebase",
				Password:             "passwd",
				Database:             "sampleDB",
				SRV:                  true,
				MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "cluster0.sample.mongodb.net",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "",
				SRV:                    true,
				MaximumSQLResultSize:   common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "cluster0.sample.mongodb.net",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "admin",
				SRV:                    true,
				MaximumSQLResultSize:   common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "node1.cluster0.sample.mongodb.net",
				Port:                   "27017",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "admin",
				SRV:                    false,
				AdditionalAddresses: []*storepb.DataSourceOptions_Address{
					{Host: "node2.cluster0.sample.mongodb.net", Port: "27017"},
					{Host: "node3.cluster0.sample.mongodb.net", Port: "27017"},
				},
				ReplicaSet:           "rs0",
				MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
			},
			want: "mongodb://bytebase:passwd@node1.cluster0.sample.mongodb.net:27017,node2.cluster0.sample.mongodb.net:27017,node3.cluster0.sample.mongodb.net:27017/sampleDB?appName=bytebase&authSource=admin&replicaSet=rs0",
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		got := getBasicMongoDBConnectionURI(tt.connConfig)
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
	}

	a := require.New(t)
	for _, tt := range tests {
		got := isMongoStatement(tt.statement)
		a.Equal(tt.want, got, tt.statement)
	}
}

func TestGetSimpleStatementResult(t *testing.T) {
	groupsValue := `[
	"basketball",
	"swimming"
]`
	treeValue := `{
	"a": "a",
	"b": 1
}`

	tests := []struct {
		data string
		want *v1pb.QueryResult
	}{
		{
			data: `[
  {
    "_id": {
      "$oid": "66f62cad7195ccc0dbdfafbb"
    },
    "a": {
      "$numberLong": "1546786128982089728"
    }
  },
  {
    "_id": {
      "$oid": "66f670827941d8cb2bac29d3"
    },
    "a": {
      "$numberLong": "1546786122282089721"
    }
  },
  {
    "_id": {
      "$oid": "66f675627ed80fb207320dd9"
    },
    "name": "danny",
    "wew": "iii"
  },
  {
    "_id": {
      "$oid": "66f6758c30daae815ac8784f"
    },
    "name": "dannyyy",
    "groups": [
      "123",
      "222"
    ]
  }
]`,
			want: &v1pb.QueryResult{
				ColumnNames:     []string{"_id", "a", "name", "wew", "groups"},
				ColumnTypeNames: []string{"TEXT", "TEXT", "TEXT", "TEXT", "TEXT"},
				Rows: []*v1pb.QueryRow{{
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "64c0b8c4e65c51195e0584b2"}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: "13"}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: groupsValue}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: `"danny"`}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: treeValue}},
					},
				}},
			},
		},
		{
			data: `
{
    "_id": {
      "$oid": "66f6758c30daae815ac8784f"
    },
    "a": {
      "$numberLong": "1546786122282089721"
    },
    "name": "dannyyy",
    "groups": [
      "123",
      "222"
    ]
}`,
			want: &v1pb.QueryResult{
				ColumnNames:     []string{"_id", "age", "flower", "groups", "name", "tree"},
				ColumnTypeNames: []string{"TEXT", "TEXT", "TEXT", "TEXT", "TEXT", "TEXT"},
				Rows: []*v1pb.QueryRow{{
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "64c0b8c4e65c51195e0584b2"}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: "13"}},
						{Kind: &v1pb.RowValue_NullValue{}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: groupsValue}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: `"danny"`}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: treeValue}},
					},
				}, {
					Values: []*v1pb.RowValue{
						{Kind: &v1pb.RowValue_StringValue{StringValue: "64c1de7e85c563e625f217d5"}},
						{Kind: &v1pb.RowValue_NullValue{}},
						{Kind: &v1pb.RowValue_StringValue{StringValue: "123"}},
						{Kind: &v1pb.RowValue_NullValue{}},
						{Kind: &v1pb.RowValue_NullValue{}},
						{Kind: &v1pb.RowValue_NullValue{}},
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
		a.Empty(diff)
	}
}

func TestGetOrderedColumns(t *testing.T) {
	tests := []struct {
		input              map[string]bool
		wantColumns        []string
		wantColumnIndexMap map[string]int
	}{
		{
			input:              map[string]bool{},
			wantColumns:        []string{},
			wantColumnIndexMap: map[string]int{},
		},
		{
			input:              map[string]bool{"_id": true},
			wantColumns:        []string{"_id"},
			wantColumnIndexMap: map[string]int{"_id": 0},
		},
		{
			input:              map[string]bool{"a": true},
			wantColumns:        []string{"a"},
			wantColumnIndexMap: map[string]int{"a": 0},
		},
		{
			input:              map[string]bool{"a": true, "_id": true, "b": true},
			wantColumns:        []string{"_id", "a", "b"},
			wantColumnIndexMap: map[string]int{"_id": 0, "a": 1, "b": 2},
		},
		{
			input:              map[string]bool{"_id": true, "a": true, "b": true},
			wantColumns:        []string{"_id", "a", "b"},
			wantColumnIndexMap: map[string]int{"_id": 0, "a": 1, "b": 2},
		},
	}
	a := require.New(t)
	for _, tt := range tests {
		gotColumns, gotMap := getOrderedColumns(tt.input)
		a.ElementsMatch(tt.wantColumns, gotColumns)
		a.Equal(tt.wantColumnIndexMap, gotMap)
	}
}

func TestUnmarshalExtJSON(t *testing.T) {
	content := []byte("[\n  {\n    \"_id\": {\n      \"$oid\": \"66f62cad7195ccc0dbdfafbb\"\n    },\n    \"a\": {\n      \"$numberLong\": \"1546786128982089728\"\n    }\n  },\n  {\n    \"_id\": {\n      \"$oid\": \"66f670827941d8cb2bac29d3\"\n    },\n    \"a\": {\n      \"$numberLong\": \"1546786122282089721\"\n    }\n  }\n]\n")

	a := require.New(t)
	var r any
	err := bson.UnmarshalExtJSON(content, true, &r)
	a.NoError(err)

	rr := r.(bson.A)

	t.Logf("%+v", rr)

	d := rr[0].(bson.D)

	for i, e := range d {
		t.Logf("%d: %v, %v, %T\n", i, e.Key, e.Value, e.Value)
	}
}
