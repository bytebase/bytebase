package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestTSQLExtractSensitiveField(t *testing.T) {
	var (
		defaultDatabase       = "MyDB"
		defaultDatabaseSchema = &db.SensitiveSchemaInfo{
			IgnoreCaseSensitive: true,
			DatabaseList: []db.DatabaseSchema{
				{
					Name: defaultDatabase,
					SchemaList: []db.SchemaSchema{
						{
							Name: "dbo",
							TableList: []db.TableSchema{
								{
									Name: "MyTable1",
									ColumnList: []db.ColumnInfo{
										{
											Name:         "a",
											MaskingLevel: storepb.MaskingLevel_FULL,
										},
										{
											Name:         "b",
											MaskingLevel: storepb.MaskingLevel_NONE,
										},
										{
											Name:         "c",
											MaskingLevel: storepb.MaskingLevel_NONE,
										},
										{
											Name:         "d",
											MaskingLevel: storepb.MaskingLevel_PARTIAL,
										},
									},
								},
								{
									Name: "MyTable2",
									ColumnList: []db.ColumnInfo{
										{
											Name:         "e",
											MaskingLevel: storepb.MaskingLevel_FULL,
										},
									},
								},
							},
						},
					},
				},
			},
		}
	)

	tests := []struct {
		statement  string
		schemaInfo *db.SensitiveSchemaInfo
		fieldList  []db.SensitiveField
	}{
		{
			// Test for recursive CTE.
			statement: `WITH cte_01 AS (
				SELECT a AS c1, b AS c2, c AS c3, 1 AS n FROM MyTable1
				UNION ALL
				SELECT c1 * c2, c2 + c1, c3 * c2, n + 1 FROM cte_01 WHERE n < 5
			)
			SELECT * FROM cte_01;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "c1",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "c2",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "c3",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "n",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
			},
		},
		// Test for multiple CTE.
		{
			statement: `
WITH tt1(aa, bb) AS (
	SELECT a, b FROM MyTable1
),
tt2(cc, dd) AS (
	SELECT c, d FROM MyTable1
),
tt3(ee) AS (
	SELECT e FROM MyTable2
)
SELECT * FROM tt1 JOIN tt2 ON tt1.aa = tt2.cc JOIN tt3 ON tt2.dd = tt3.ee;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "aa",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "bb",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "cc",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "dd",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
				{
					Name:         "ee",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
		},
		// Test for CTE.
		{
			statement: `
WITH tt1(aa, bb) AS (
	SELECT a, b FROM MyTable1
)
SELECT tt1.aa, bb FROM tt1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "aa",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "bb",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
			},
		},
		// Test for subquery in from cluase with as alias.
		{
			statement:  `SELECT tt.a, b FROM (SELECT * FROM MyTable1) AS tt`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "a",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "b",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
			},
		},
		// Table source list.
		{
			statement:  `SELECT a, b, c, d, e FROM MyTable1, MyTable2;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "a",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "b",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "c",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "d",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
				{
					Name:         "e",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
		},
		// Join
		{
			statement:  `SELECT a, b, c, d, e FROM MyTable1 JOIN MyTable2 ON MyTable1.a = MyTable2.e;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "a",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "b",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "c",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "d",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
				{
					Name:         "e",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
		},
		// Union
		{
			statement:  `SELECT b FROM MyTable1 UNION SELECT e FROM MyTable2;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "b",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
		},
		// Subquery in Select list.
		{
			statement:  `SELECT (SELECT MAX(e) FROM MyTable2) FROM MyTable1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "SELECTMAX(e)FROMMyTable2",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
		},
		// Alias table source.
		{
			statement:  `SELECT T1.a FROM MyTable1 AS T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "a",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
		},
		// Asterisk in SELECT list.
		{
			statement:  `SELECT * FROM MyTable1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:         "a",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
				{
					Name:         "b",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "c",
					MaskingLevel: storepb.MaskingLevel_NONE,
				},
				{
					Name:         "d",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
			},
		},
	}

	for _, test := range tests {
		res, err := GetMaskedFields(test.statement, defaultDatabase, test.schemaInfo)
		require.NoError(t, err, test.statement)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}
