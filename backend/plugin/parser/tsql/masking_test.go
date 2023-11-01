package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTSQLExtractSensitiveField(t *testing.T) {
	var (
		defaultDatabase       = "MyDB"
		defaultDatabaseSchema = &base.SensitiveSchemaInfo{
			IgnoreCaseSensitive: true,
			DatabaseList: []base.DatabaseSchema{
				{
					Name: defaultDatabase,
					SchemaList: []base.SchemaSchema{
						{
							Name: "dbo",
							TableList: []base.TableSchema{
								{
									Name: "MyTable1",
									ColumnList: []base.ColumnInfo{
										{
											Name:              "a",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
										},
										{
											Name:              "b",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
										},
										{
											Name:              "c",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
										},
										{
											Name:              "d",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
										},
									},
								},
								{
									Name: "MyTable2",
									ColumnList: []base.ColumnInfo{
										{
											Name:              "e",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
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
		schemaInfo *base.SensitiveSchemaInfo
		fieldList  []base.SensitiveField
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
			fieldList: []base.SensitiveField{
				{
					Name:              "c1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "c2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "c3",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "n",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
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
			fieldList: []base.SensitiveField{
				{
					Name:              "aa",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "bb",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "cc",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "dd",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
				{
					Name:              "ee",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
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
			fieldList: []base.SensitiveField{
				{
					Name:              "aa",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "bb",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		// Test for subquery in from cluase with as alias.
		{
			statement:  `SELECT tt.a, b FROM (SELECT * FROM MyTable1) AS tt`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		// Table source list.
		{
			statement:  `SELECT a, b, c, d, e FROM MyTable1, MyTable2;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
				{
					Name:              "e",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		// Join
		{
			statement:  `SELECT a, b, c, d, e FROM MyTable1 JOIN MyTable2 ON MyTable1.a = MyTable2.e;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
				{
					Name:              "e",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		// Union
		{
			statement:  `SELECT b FROM MyTable1 UNION SELECT e FROM MyTable2;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		// Subquery in Select list.
		{
			statement:  `SELECT (SELECT MAX(e) FROM MyTable2) FROM MyTable1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "SELECTMAX(e)FROMMyTable2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		// Alias table source.
		{
			statement:  `SELECT T1.a FROM MyTable1 AS T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		// Asterisk in SELECT list.
		{
			statement:  `SELECT * FROM MyTable1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
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
