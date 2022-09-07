package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor/db"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

type testData struct {
	origin    *Database
	statement string
	want      *Database
}

var (
	one = "1"
)

func TestWalkThrough(t *testing.T) {
	tests := []testData{
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci,
					KEY idx_a (a),
					INDEX (b, a),
					UNIQUE (b, c, d),
					FULLTEXT (b, d) WITH PARSER ngram
				)
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name: "t",
								ColumnList: []*Column{
									{
										Name:     "a",
										Position: 1,
										Default:  &one,
										Nullable: false,
										Type:     "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Default:      nil,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:     "c",
										Position: 3,
										Default:  nil,
										Nullable: true,
										Type:     "int(11)",
										Comment:  "This is a comment",
									},
									{
										Name:      "d",
										Position:  4,
										Default:   nil,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
								},
								IndexList: []*Index{
									{
										Name:           "PRIMARY",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        true,
									},
									{
										Name:           "b",
										ExpressionList: []string{"b"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
									},
									{
										Name:           "idx_a",
										ExpressionList: []string{"a"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
									},
									{
										Name:           "b_2",
										ExpressionList: []string{"b", "a"},
										Type:           "BTREE",
										Unique:         false,
										Primary:        false,
									},
									{
										Name:           "b_3",
										ExpressionList: []string{"b", "c", "d"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        false,
									},
									{
										Name:           "b_4",
										ExpressionList: []string{"b", "d"},
										Type:           "FULLTEXT",
										Unique:         false,
										Primary:        false,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &Database{
				Name:   "test",
				DbType: db.MySQL,
			},
			statement: `
				CREATE TABLE t(
					a int,
					b int,
					PRIMARY KEY (a, b)
				)
			`,
			want: &Database{
				Name:   "test",
				DbType: db.MySQL,
				SchemaList: []*Schema{
					{
						Name: "",
						TableList: []*Table{
							{
								Name: "t",
								ColumnList: []*Column{
									{
										Name:     "a",
										Position: 1,
										Type:     "int(11)",
										Nullable: false,
									},
									{
										Name:     "b",
										Position: 2,
										Type:     "int(11)",
										Nullable: false,
									},
								},
								IndexList: []*Index{
									{
										Name:           "PRIMARY",
										ExpressionList: []string{"a", "b"},
										Type:           "BTREE",
										Unique:         true,
										Primary:        true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		state := newDatabaseState(test.origin)
		err := state.WalkThrough(test.statement)
		require.NoError(t, err)
		want := newDatabaseState(test.want)
		require.Equal(t, want, state, test.statement)
	}
}
