package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSnowSQLExtractSensitiveField(t *testing.T) {
	var (
		defaultDatabase       = "SNOWFLAKE"
		defaultDatabaseSchema = &base.SensitiveSchemaInfo{
			DatabaseList: []base.DatabaseSchema{
				{
					Name: defaultDatabase,
					SchemaList: []base.SchemaSchema{
						{
							Name: "PUBLIC",
							TableList: []base.TableSchema{
								{
									Name: "T1",
									ColumnList: []base.ColumnInfo{
										{
											Name:              "A",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
										},
										{
											Name:              "B",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
										},
										{
											Name:              "C",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
										},
										{
											Name:              "D",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
										},
									},
								},
								{
									Name: "T2",
									ColumnList: []base.ColumnInfo{
										{
											Name:              "A",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
										},
										{
											Name:              "E",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
										},
									},
								},
								{
									Name: "T3",
									ColumnList: []base.ColumnInfo{
										{
											Name:              "E",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
										},
										{
											Name:              "F",
											MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
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
			statement: `WITH CTE_01 AS (
				SELECT A AS C1, B AS C2, C AS C3, 1 AS N FROM T1
				UNION ALL
				SELECT C1 * C2, C2 + C1, C3 * C2, N + 1 FROM CTE_01 WHERE N < 5
			)
			SELECT * FROM CTE_01;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "C1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C3",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "N",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for UNPIVOT.
			statement:  `SELECT * FROM T1 UNPIVOT(E FOR F IN (B, C, D));`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "F",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "E",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for PIVOT.
			statement:  `SELECT TT1.* FROM T1 PIVOT(MAX(A) FOR B IN ('a', 'b', 'c')) AS TT1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "C",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "D",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
				{
					Name:              `'a'`,
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              `'b'`,
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              `'c'`,
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for correlated sub-query.
			statement:  `SELECT A, (SELECT MAX(B) > Y.A FROM T1 X) FROM T1 Y`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "(SELECTMAX(B)>Y.AFROMT1X)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for CTE in CTE.
			statement: `WITH TT1 (T1_COL1, T1_COL2) AS (
				WITH TT2 (T1_COL1, T1_COL2, T1_COL3) AS (
					SELECT A, B, C FROM T1
				)
				SELECT T1_COL1, T1_COL2 FROM TT2
			)
			SELECT * FROM TT1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "T1_COL1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "T1_COL2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for expression.
			statement:  `SELECT (SELECT A FROM T1 LIMIT 1), A + 1, 1, FUNCTIONCALL(D) FROM T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "(SELECTAFROMT1LIMIT1)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "A+1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "FUNCTIONCALL(D)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for multiple CTE
			statement: `
			WITH TT1 (T1_COL1, T1_COL2, T1_COL3, T1_COL4) AS (
				SELECT * FROM T1
			),
			TT2 (T2_COL1, T2_COL2) AS (
				SELECT * FROM T2
			)
			SELECT * FROM TT1 JOIN TT2;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "T1_COL1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "T1_COL2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "T1_COL3",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "T1_COL4",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
				{
					Name:              "T2_COL1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "T2_COL2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for set operators(UNION, INTERSECT, ...)
			statement:  `SELECT A, B FROM T1 UNION SELECT * FROM T2 INTERSECT SELECT * FROM T3`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for subquery in from cluase with as alias.
			statement:  `SELECT T.A, A, B FROM (SELECT * FROM T1) AS T`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for field name.
			statement:  "SELECT $1, A, T.B AS N, T.C from T1 AS T",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "N",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "C",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			statement:  `SELECT * FROM T1, T2, T3;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "C",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "D",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "E",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "E",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "F",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			statement:  `SELECT A, E, F FROM T1 NATURAL JOIN T2 NATURAL JOIN T3;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "E",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "F",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			statement:  `SELECT A, B, D FROM T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "D",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
			},
		},
		{
			statement:  `SELECT * FROM T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "C",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "D",
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
