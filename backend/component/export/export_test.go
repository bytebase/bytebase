package export

import (
	"testing"

	"github.com/stretchr/testify/assert"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetSQLStatementPrefix(t *testing.T) {
	tests := []struct {
		engine       storepb.Engine
		resourceList []base.SchemaResource
		columnNames  []string
		want         string
	}{
		{
			engine:       storepb.Engine_MYSQL,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO `<table_name>` (`a`) VALUES (",
		},
		{
			engine:       storepb.Engine_MYSQL,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a", "b"},
			want:         "INSERT INTO `table1` (`a`,`b`) VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"<table_name>\" (\"a\") VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"table1\" (\"a\") VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "schema1", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"schema1\".\"table1\" (\"a\") VALUES (",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		got, err := SQLStatementPrefix(test.engine, test.resourceList, test.columnNames)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestExportSQL(t *testing.T) {
	tests := []struct {
		engine          storepb.Engine
		statementPrefix string
		result          *v1pb.QueryResult
		want            string
	}{
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BoolValue{BoolValue: true},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "abc"},
							},
							{
								Kind: &v1pb.RowValue_NullValue{},
							},
						},
					},
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BoolValue{BoolValue: false},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "abc"},
							},
							{
								Kind: &v1pb.RowValue_NullValue{},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES (true,'abc',NULL);\nINSERT INTO `<table_name>` (`a`) VALUES (false,'abc',NULL);",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\nbc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\\nbc');",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a'b"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a''b');",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\b"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\\b');",
		},
		{
			engine:          storepb.Engine_POSTGRES,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\nbc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\nbc');",
		},
		{
			engine:          storepb.Engine_POSTGRES,
			statementPrefix: "INSERT INTO `<table_name>` (`b`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\\bc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`b`) VALUES ( E'a\\\\bc');",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte{0b101}},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES (0x05);",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		got, err := SQL(test.engine, test.statementPrefix, test.result)
		a.NoError(err)
		a.Equal(test.want, string(got))
	}
}

func TestGetExcelColumnName(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		index int
		want  string
	}{
		{
			index: 0,
			want:  "A",
		},
		{
			index: 3,
			want:  "D",
		},
		{
			index: 25,
			want:  "Z",
		},
		{
			index: 26,
			want:  "AA",
		},
		{
			index: 27,
			want:  "AB",
		},
		{
			index: ExcelMaxColumn - 1,
			want:  "ZZZ",
		},
	}

	for _, test := range tests {
		got, err := ExcelColumnName(test.index)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestExportJSON(t *testing.T) {
	tests := []struct {
		result *v1pb.QueryResult
		want   string
	}{
		{
			result: &v1pb.QueryResult{
				ColumnNames: []string{"a"},
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte{0b101}},
							},
						},
					},
				},
			},
			want: `[
  {
    "a": "BQ=="
  }
]`,
		},
		{
			result: &v1pb.QueryResult{
				ColumnNames: []string{"id", "name", "email", "age"},
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_Int32Value{Int32Value: 1},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "Alice"},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a@bytebase.com"},
							},
							{
								Kind: &v1pb.RowValue_Int32Value{Int32Value: 20},
							},
						},
					},
				},
			},
			want: `[
  {
    "age": 20,
    "email": "a@bytebase.com",
    "id": 1,
    "name": "Alice"
  }
]`,
		},
	}

	a := assert.New(t)
	for _, test := range tests {
		got, err := JSON(test.result)
		a.NoError(err)
		a.Equal(test.want, string(got))
	}
}
