package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPLSQLExtractSensitiveField(t *testing.T) {
	const (
		defaultSchema = "ROOT"
	)
	var (
		defaultDatabaseSchema = &base.SensitiveSchemaInfo{
			DatabaseList: []base.DatabaseSchema{
				{
					Name: defaultSchema,
					SchemaList: []base.SchemaSchema{
						{
							Name: defaultSchema,
							TableList: []base.TableSchema{
								{
									Name: "T",
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
			// Test for Recursive Common Table Expression dependent closures.
			statement: `
				with t1(cc1, cc2, cc3, n) as (
					select a as c1, b as c2, c as c3, 1 as n from t
					union all
					select cc1 * cc2, cc2 + cc1, cc3 * cc2, n + 1 from t1 where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "CC1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "CC2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "CC3",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "N",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for Recursive Common Table Expression.
			statement: `
				with t1 as (
					select 1 as c1, 2 as c2, 3 as c3, 1 as n from DUAL
					union all
					select c1 * a, c2 * b, c3 * d, n + 1 from t1, t where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "C1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
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
			// Test that Common Table Expression rename field names.
			statement:  `with t1(d, c, b, a) as (select * from t) select * from t1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "D",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultRangeMasker()),
				},
			},
		},
		{
			// Test for Common Table Expression with UNION.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from (select * from t1 union all select * from t2)`,
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
		{
			// Test for Common Table Expression reference.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from t2`,
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
		{
			// Test for multi-level Common Table Expression.
			statement:  `with tt2 as (with tt2 as (select * from t) select MAX(A) from tt2) select * from tt2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "MAX(A)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for Common Table Expression.
			statement:  `with t1 as (select * from t) select * from t1`,
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
		{
			// Test for UNION.
			statement:  `select 1 as c1, 2 as c2, 3 as c3, 4 from DUAL UNION ALL select * from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "C1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C2",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "C3",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
				{
					Name:              "4",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for UNION.
			statement:  `select * from t UNION ALL select * from t`,
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
		{
			// Test for explicit schema name.
			statement:  `select CONCAT(ROOT.T.A, ROOT.T.B) from T`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "CONCAT(ROOT.T.A,ROOT.T.B)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for associated sub-query.
			statement:  `select a, (SELECT MAX(B) > Y.A FROM T X) from t y`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "(SELECTMAX(B)>Y.AFROMTX)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for JOIN with ON clause.
			statement:  `select * from t t1 join t t2 on t1.a = t2.a`,
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
			// Test for natural JOIN.
			statement:  `select * from t t1 natural join t t2`,
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
		{
			// Test for JOIN with USING clause.
			statement:  `select * from t t1 join t t2 using(a)`,
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
			// Test for non-associated sub-query
			statement:  "select t.a, (SELECT MAX(A) FROM T) from t t1 join t on t.a = t1.b",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "(SELECTMAX(A)FROMT)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for functions.
			statement:  `select A-B, B+C as c1 from (select * from t)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "A-B",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for functions.
			statement:  `select MAX(A), min(b) as c1 from (select * from t)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "MAX(A)",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
				{
					Name:              "C1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker()),
				},
			},
		},
		{
			// Test for sub-query
			statement:  "select * from (select * from t) where rownum <= 100000;",
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
		{
			// Test for sub-select.
			statement:  "select * from (select a, t.b, root.t.c, d as d1 from root.t) where ROWNUM <= 100000;",
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
					Name:              "D1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			// Test for field name.
			statement:  "select a, t.b, root.t.c, d as d1 from t",
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
					Name:              "D1",
					MaskingAttributes: base.NewMaskingAttributes(masker.NewDefaultFullMasker()),
				},
			},
		},
		{
			statement:  "SELECT * FROM ROOT.T;",
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
		{
			// Test for EXPLAIN statements.
			statement:  "explain plan for select 1 from dual;",
			schemaInfo: &base.SensitiveSchemaInfo{},
			fieldList:  nil,
		},
		{
			// Test for no FROM DUAL.
			statement:  "select 1 from dual;",
			schemaInfo: &base.SensitiveSchemaInfo{},
			fieldList:  []base.SensitiveField{{Name: "1", MaskingAttributes: base.NewMaskingAttributes(masker.NewNoneMasker())}},
		},
	}

	for _, test := range tests {
		res, err := GetMaskedFields(test.statement, defaultSchema, test.schemaInfo)
		require.NoError(t, err, test.statement)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}
