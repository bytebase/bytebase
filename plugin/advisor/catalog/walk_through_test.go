package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

type testData struct {
	origin    *storepb.DatabaseMetadata
	statement string
	want      *storepb.DatabaseMetadata
	err       error
}

var (
	one = "1"
)

func TestWalkThrough(t *testing.T) {
	tests := []testData{
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a TEXT);
				ALTER TABLE t ALTER COLUMN a SET DEFAULT '1';
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeInvalidColumnTypeForDefaultValue,
				Content: "BLOB, TEXT, GEOMETRY or JSON column `a` can't have a default value",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a TEXT DEFAULT '1')
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeInvalidColumnTypeForDefaultValue,
				Content: "BLOB, TEXT, GEOMETRY or JSON column `a` can't have a default value",
				Line:    2,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a INT NOT NULL);
				ALTER TABLE t ALTER COLUMN a SET DEFAULT NULL;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeSetNullDefaultForNotNullColumn,
				Content: "Invalid default value for column `a`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a INT NOT NULL DEFAULT NULL)
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeSetNullDefaultForNotNullColumn,
				Content: "Invalid default value for column `a`",
				Line:    2,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int ON UPDATE NOW())
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeOnUpdateColumnNotDatetimeOrTimestamp,
				Content: "Column `a` use ON UPDATE but is not DATETIME or TIMESTAMP",
				Line:    2,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int auto_increment, b int auto_increment);
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeAutoIncrementExists,
				Content: "There can be only one auto column for table `t`",
				Line:    2,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t as select * from t1;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeUseCreateTableAs,
				Content: "Disallow the CREATE TABLE AS statement but \"CREATE TABLE t as select * from t1;\" uses",
				Line:    2,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				CREATE INDEX idx_c on t(c);
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				ALTER TABLE t MODIFY COLUMN c int;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				ALTER TABLE t CHANGE COLUMN c aa int;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				ALTER TABLE t DROP COLUMN c;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				ALTER TABLE t RENAME COLUMN c TO cc;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},

		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				ALTER TABLE t RENAME COLUMN c TO cc;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int, b int); 
				ALTER TABLE t ALTER COLUMN c DROP DEFAULT;
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: "Column `c` does not exist in table `t`",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				ALTER DATABASE CHARACTER SET = utf8mb4;
				ALTER DATABASE test COLLATE utf8mb4_polish_ci;
			`,
			want: &storepb.DatabaseMetadata{
				Name:         "test",
				CharacterSet: "utf8mb4",
				Collation:    "utf8mb4_polish_ci",
				Schemas:      []*storepb.SchemaMetadata{{}},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment' DEFAULT NULL,
					d varchar(10) COLLATE utf8mb4_polish_ci,
					KEY idx_a (a),
					INDEX (b, a),
					UNIQUE (b, c, d),
					FULLTEXT (b, d) WITH PARSER ngram INVISIBLE
				);
				CREATE TABLE t_copy like t;
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t_copy",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:       "a",
										Position:   1,
										HasDefault: true,
										Default:    one,
										Nullable:   false,
										Type:       "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										HasDefault:   false,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:       "c",
										Position:   3,
										HasDefault: false,
										Nullable:   true,
										Type:       "int(11)",
										Comment:    "This is a comment",
									},
									{
										Name:       "d",
										Position:   4,
										HasDefault: false,
										Nullable:   true,
										Type:       "varchar(10)",
										Collation:  "utf8mb4_polish_ci",
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "PRIMARY",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     true,
										Visible:     true,
									},
									{
										Name:        "b",
										Expressions: []string{"b"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "idx_a",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_2",
										Expressions: []string{"b", "a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_3",
										Expressions: []string{"b", "c", "d"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_4",
										Expressions: []string{"b", "d"},
										Type:        "FULLTEXT",
										Unique:      false,
										Primary:     false,
										Visible:     false,
									},
								},
							},
							{
								Name: "t",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:       "a",
										Position:   1,
										HasDefault: true,
										Default:    one,
										Nullable:   false,
										Type:       "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:     "c",
										Position: 3,
										Nullable: true,
										Type:     "int(11)",
										Comment:  "This is a comment",
									},
									{
										Name:      "d",
										Position:  4,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "PRIMARY",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     true,
										Visible:     true,
									},
									{
										Name:        "b",
										Expressions: []string{"b"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "idx_a",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_2",
										Expressions: []string{"b", "a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_3",
										Expressions: []string{"b", "c", "d"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_4",
										Expressions: []string{"b", "d"},
										Type:        "FULLTEXT",
										Unique:      false,
										Primary:     false,
										Visible:     false,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
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
					FULLTEXT (b, d) WITH PARSER ngram INVISIBLE
				)
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:       "a",
										Position:   1,
										HasDefault: true,
										Default:    one,
										Nullable:   false,
										Type:       "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:     "c",
										Position: 3,
										Nullable: true,
										Type:     "int(11)",
										Comment:  "This is a comment",
									},
									{
										Name:      "d",
										Position:  4,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "PRIMARY",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     true,
										Visible:     true,
									},
									{
										Name:        "b",
										Expressions: []string{"b"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "idx_a",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_2",
										Expressions: []string{"b", "a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_3",
										Expressions: []string{"b", "c", "d"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_4",
										Expressions: []string{"b", "d"},
										Type:        "FULLTEXT",
										Unique:      false,
										Primary:     false,
										Visible:     false,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(
					a int,
					b int,
					PRIMARY KEY (a, b)
				)
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t",
								Columns: []*storepb.ColumnMetadata{
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
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "PRIMARY",
										Expressions: []string{"a", "b"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     true,
										Visible:     true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t1(a int, b int, c int);
				CREATE TABLE t2(a int);
				DROP TABLE t1, t2
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Tables: []*storepb.TableMetadata{},
						Views:  []*storepb.ViewMetadata{},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				DROP TABLE t1, t2
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeTableNotExists,
				Content: "Table `t1` does not exist",
				Line:    2,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int);
				RENAME TABLE t to other_db.t1
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(a int);
				RENAME TABLE t to test.t1
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t1",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "a",
										Position: 1,
										Nullable: true,
										Type:     "int(11)",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `-- this is comment
				DROP DATABASE test;
				CREATE TABLE t(a int);
			`,
			err: &WalkThroughError{
				Type:    ErrorTypeDatabaseIsDeleted,
				Content: "Database `test` is deleted",
				Line:    3,
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci,
					e int,
					KEY idx_a (a),
					INDEX (b, a),
					UNIQUE (b, c, d),
					FULLTEXT (b, d) WITH PARSER ngram INVISIBLE
				);
				ALTER TABLE t COLLATE utf8mb4_0900_ai_ci, ENGINE = INNODB, COMMENT 'This is a table comment';
				ALTER TABLE t ADD COLUMN a1 int AFTER a;
				ALTER TABLE t ADD INDEX idx_a_b (a, b);
				ALTER TABLE t DROP COLUMN c;
				ALTER TABLE t DROP PRIMARY KEY;
				ALTER TABLE t DROP INDEX b_2;
				ALTER TABLE t MODIFY COLUMN b varchar(20) FIRST;
				ALTER TABLE t CHANGE COLUMN d d_copy varchar(10) COLLATE utf8mb4_polish_ci;
				ALTER TABLE t RENAME COLUMN a to a_copy;
				ALTER TABLE t RENAME TO t_copy;
				ALTER TABLE t_copy ALTER COLUMN a_copy DROP DEFAULT;
				ALTER TABLE t_copy RENAME INDEX b TO idx_b;
				ALTER TABLE t_copy ALTER INDEX b_3 INVISIBLE;
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name:      "t_copy",
								Collation: "utf8mb4_0900_ai_ci",
								Engine:    "INNODB",
								Comment:   "This is a table comment",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "b",
										Position: 1,
										Nullable: true,
										Type:     "varchar(20)",
									},
									{
										Name:     "a_copy",
										Position: 2,
										Nullable: false,
										Type:     "int(11)",
									},
									{
										Name:     "a1",
										Position: 3,
										Nullable: true,
										Type:     "int(11)",
									},

									{
										Name:      "d_copy",
										Position:  4,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
									{
										Name:     "e",
										Position: 5,
										Nullable: true,
										Type:     "int(11)",
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "idx_b",
										Expressions: []string{"b"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "idx_a",
										Expressions: []string{"a_copy"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_3",
										Expressions: []string{"b", "d_copy"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     false,
									},
									{
										Name:        "b_4",
										Expressions: []string{"b", "d_copy"},
										Type:        "FULLTEXT",
										Unique:      false,
										Primary:     false,
										Visible:     false,
									},
									{
										Name:        "idx_a_b",
										Expressions: []string{"a_copy", "b"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci
				);
				CREATE INDEX idx_a on t(a);
				CREATE INDEX b_2 on t(b, a);
				CREATE UNIQUE INDEX b_3 on t(b, c, d);
				CREATE FULLTEXT INDEX b_4 on t(b, d) WITH PARSER ngram INVISIBLE;
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:       "a",
										Position:   1,
										HasDefault: true,
										Default:    one,
										Nullable:   false,
										Type:       "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
									{
										Name:     "c",
										Position: 3,
										Nullable: true,
										Type:     "int(11)",
										Comment:  "This is a comment",
									},
									{
										Name:      "d",
										Position:  4,
										Nullable:  true,
										Type:      "varchar(10)",
										Collation: "utf8mb4_polish_ci",
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "PRIMARY",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     true,
										Visible:     true,
									},
									{
										Name:        "b",
										Expressions: []string{"b"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "idx_a",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_2",
										Expressions: []string{"b", "a"},
										Type:        "BTREE",
										Unique:      false,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_3",
										Expressions: []string{"b", "c", "d"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     false,
										Visible:     true,
									},
									{
										Name:        "b_4",
										Expressions: []string{"b", "d"},
										Type:        "FULLTEXT",
										Unique:      false,
										Primary:     false,
										Visible:     false,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			origin: &storepb.DatabaseMetadata{
				Name: "test",
			},
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE
				);
				DROP INDEX b on t;
			`,
			want: &storepb.DatabaseMetadata{
				Name: "test",

				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:       "a",
										Position:   1,
										HasDefault: true,
										Default:    one,
										Nullable:   false,
										Type:       "int(11)",
									},
									{
										Name:         "b",
										Position:     2,
										Nullable:     false,
										Type:         "varchar(200)",
										CharacterSet: "utf8mb4",
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "PRIMARY",
										Expressions: []string{"a"},
										Type:        "BTREE",
										Unique:      true,
										Primary:     true,
										Visible:     true,
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
		state := newDatabaseState(test.origin, &FinderContext{CheckIntegrity: true, EngineType: db.MySQL})
		err := state.WalkThrough(test.statement)
		if test.err != nil {
			require.Equal(t, err, test.err)
			continue
		}
		require.NoError(t, err)
		want := newDatabaseState(test.want, &FinderContext{CheckIntegrity: true, EngineType: db.MySQL})
		require.Equal(t, want, state, test.statement)
	}
}

type testDataForIncomplete struct {
	statement string
	want      *DatabaseState
	err       error
}

func TestWalkThroughForIncompleteOriginalCatalog(t *testing.T) {
	tests := []testDataForIncomplete{
		{
			statement: `CREATE INDEX idx_a on t(a)`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: "",
				collation:    "",
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:  &FinderContext{CheckIntegrity: false},
						name: "",
						tableSet: tableStateMap{
							"t": {
								name:      "t",
								engine:    nil,
								collation: nil,
								comment:   nil,
								columnSet: columnStateMap{},
								indexSet: indexStateMap{
									"idx_a": {
										name:           "idx_a",
										expressionList: []string{"a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
								},
							},
						},
						viewSet: viewStateMap{},
					},
				},
			},
		},
		{
			statement: `ALTER TABLE t RENAME COLUMN a TO a_copy`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: *newEmptyStringPointer(),
				collation:    *newEmptyStringPointer(),
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:  &FinderContext{CheckIntegrity: false},
						name: "",
						tableSet: tableStateMap{
							"t": {
								name:      "t",
								engine:    nil,
								collation: nil,
								comment:   nil,
								columnSet: columnStateMap{
									"a_copy": {
										name: "a_copy",
									},
								},
								indexSet: indexStateMap{},
							},
						},
						viewSet: viewStateMap{},
					},
				},
			},
		},
		{
			statement: `ALTER TABLE t RENAME TO t1`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: "",
				collation:    "",
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:  &FinderContext{CheckIntegrity: false},
						name: "",
						tableSet: tableStateMap{
							"t1": {
								name:      "t1",
								engine:    nil,
								collation: nil,
								comment:   nil,
								columnSet: columnStateMap{},
								indexSet:  indexStateMap{},
							},
						},
						viewSet: viewStateMap{},
					},
				},
			},
		},
		{
			statement: `
				ALTER TABLE t ADD PRIMARY KEY (a);
				ALTER TABLE t ADD UNIQUE (b);
				CREATE INDEX idx_a on t(a);
				CREATE INDEX b_2 on t(b, a);
				CREATE UNIQUE INDEX b_3 on t(b, c, d);
				CREATE FULLTEXT INDEX b_4 on t(b, d) WITH PARSER ngram INVISIBLE;
			`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: "",
				collation:    "",
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:  &FinderContext{CheckIntegrity: false},
						name: "",
						tableSet: tableStateMap{
							"t": {
								name:      "t",
								engine:    nil,
								collation: nil,
								comment:   nil,
								columnSet: columnStateMap{},
								indexSet: indexStateMap{
									"PRIMARY": {
										name:           "PRIMARY",
										expressionList: []string{"a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newTruePointer(),
										primary:        newTruePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b": {
										name:           "b",
										expressionList: []string{"b"},
										indextype:      newStringPointer("BTREE"),
										unique:         newTruePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"idx_a": {
										name:           "idx_a",
										expressionList: []string{"a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b_2": {
										name:           "b_2",
										expressionList: []string{"b", "a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b_3": {
										name:           "b_3",
										expressionList: []string{"b", "c", "d"},
										indextype:      newStringPointer("BTREE"),
										unique:         newTruePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b_4": {
										name:           "b_4",
										expressionList: []string{"b", "d"},
										indextype:      newStringPointer("FULLTEXT"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newFalsePointer(),
										comment:        newEmptyStringPointer(),
									},
								},
							},
						},
						viewSet: viewStateMap{},
					},
				},
			},
		},
		{
			statement: `
				CREATE TABLE t(
					a int PRIMARY KEY DEFAULT 1,
					b varchar(200) CHARACTER SET utf8mb4 NOT NULL UNIQUE,
					c int auto_increment NULL COMMENT 'This is a comment',
					d varchar(10) COLLATE utf8mb4_polish_ci,
					KEY idx_a (a),
					INDEX (b, a),
					UNIQUE (b, c, d),
					FULLTEXT (b, d) WITH PARSER ngram INVISIBLE
				)
			`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: "",
				collation:    "",
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:  &FinderContext{CheckIntegrity: false},
						name: "",
						tableSet: tableStateMap{
							"t": {
								name:      "t",
								engine:    newEmptyStringPointer(),
								collation: newEmptyStringPointer(),
								comment:   newEmptyStringPointer(),
								columnSet: columnStateMap{
									"a": {
										name:         "a",
										position:     newIntPointer(1),
										defaultValue: &one,
										nullable:     newFalsePointer(),
										columnType:   newStringPointer("int(11)"),
										characterSet: newEmptyStringPointer(),
										collation:    newEmptyStringPointer(),
										comment:      newEmptyStringPointer(),
									},
									"b": {
										name:         "b",
										position:     newIntPointer(2),
										defaultValue: nil,
										nullable:     newFalsePointer(),
										columnType:   newStringPointer("varchar(200)"),
										characterSet: newStringPointer("utf8mb4"),
										collation:    newEmptyStringPointer(),
										comment:      newEmptyStringPointer(),
									},
									"c": {
										name:         "c",
										position:     newIntPointer(3),
										defaultValue: nil,
										nullable:     newTruePointer(),
										columnType:   newStringPointer("int(11)"),
										characterSet: newEmptyStringPointer(),
										collation:    newEmptyStringPointer(),
										comment:      newStringPointer("This is a comment"),
									},
									"d": {
										name:         "d",
										position:     newIntPointer(4),
										defaultValue: nil,
										nullable:     newTruePointer(),
										columnType:   newStringPointer("varchar(10)"),
										characterSet: newEmptyStringPointer(),
										collation:    newStringPointer("utf8mb4_polish_ci"),
										comment:      newEmptyStringPointer(),
									},
								},
								indexSet: indexStateMap{
									"PRIMARY": {
										name:           "PRIMARY",
										expressionList: []string{"a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newTruePointer(),
										primary:        newTruePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b": {
										name:           "b",
										expressionList: []string{"b"},
										indextype:      newStringPointer("BTREE"),
										unique:         newTruePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"idx_a": {
										name:           "idx_a",
										expressionList: []string{"a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b_2": {
										name:           "b_2",
										expressionList: []string{"b", "a"},
										indextype:      newStringPointer("BTREE"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b_3": {
										name:           "b_3",
										expressionList: []string{"b", "c", "d"},
										indextype:      newStringPointer("BTREE"),
										unique:         newTruePointer(),
										primary:        newFalsePointer(),
										visible:        newTruePointer(),
										comment:        newEmptyStringPointer(),
									},
									"b_4": {
										name:           "b_4",
										expressionList: []string{"b", "d"},
										indextype:      newStringPointer("FULLTEXT"),
										unique:         newFalsePointer(),
										primary:        newFalsePointer(),
										visible:        newFalsePointer(),
										comment:        newEmptyStringPointer(),
									},
								},
							},
						},
						viewSet: viewStateMap{},
					},
				},
			},
		},
		{
			statement: `DROP TABLE t1, t2`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: "",
				collation:    "",
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:      &FinderContext{CheckIntegrity: false},
						name:     "",
						tableSet: tableStateMap{},
						viewSet:  viewStateMap{},
					},
				},
			},
		},
		{
			statement: `INSERT INTO test values (1)`,
			want: &DatabaseState{
				ctx:          &FinderContext{CheckIntegrity: false},
				name:         "",
				characterSet: "",
				collation:    "",
				dbType:       db.MySQL,
				schemaSet: schemaStateMap{
					"": {
						ctx:      &FinderContext{CheckIntegrity: false},
						name:     "",
						tableSet: tableStateMap{},
						viewSet:  viewStateMap{},
					},
				},
			},
		},
	}
	for _, test := range tests {
		finder := NewEmptyFinder(&FinderContext{CheckIntegrity: false, EngineType: db.MySQL})
		err := finder.WalkThrough(test.statement)
		if test.err != nil {
			require.Equal(t, err, test.err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, test.want, finder.Final, test.statement)
	}
}
