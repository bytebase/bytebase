package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestPostgreSQLExtractSensitiveField(t *testing.T) {
	const (
		defaultDatabase = "db"
	)
	var (
		defaultDatabaseSchema = &base.SensitiveSchemaInfo{
			DatabaseList: []base.DatabaseSchema{
				{
					Name: defaultDatabase,
					SchemaList: []base.SchemaSchema{
						{
							Name: "public",
							TableList: []base.TableSchema{
								{
									Name: "t",
									ColumnList: []base.ColumnInfo{
										{
											Name:              "a",
											MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
										},
										{
											Name:              "b",
											MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
										},
										{
											Name:              "c",
											MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
										},
										{
											Name:              "d",
											MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
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
			// Test for Non-Recursive Common Table Expression with RECURSIVE key words.
			statement: `
				with recursive t1 as (
					select 1 as c1, 2 as c2, 3 as c3, 1 as n
					union
					select a, b, d, c from t
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "c1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "c2",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c3",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
				{
					Name:              "n",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
			},
		},
		{
			// Test for Recursive Common Table Expression dependent closures.
			statement: `
				with recursive t1(cc1, cc2, cc3, n) as (
					select a as c1, b as c2, c as c3, 1 as n from t
					union
					select cc1 * cc2, cc2 + cc1, cc3 * cc2, n + 1 from t1 where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "cc1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "cc2",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "cc3",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "n",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
			},
		},
		{
			// Test for Recursive Common Table Expression.
			statement: `
				with recursive t1 as (
					select 1 as c1, 2 as c2, 3 as c3, 1 as n
					union
					select c1 * a, c2 * b, c3 * d, n + 1 from t1, t where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "c1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "c2",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c3",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
				{
					Name:              "n",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
			},
		},
		{
			// Test that Common Table Expression rename field names.
			statement:  `with t1(d, c, b, a) as (select * from t) select * from t1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for Common Table Expression with UNION.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from t1 union all select * from t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for Common Table Expression reference.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for multi-level Common Table Expression.
			statement:  `with tt2 as (with tt2 as (select * from t) select max(a) from tt2) select * from tt2;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "max",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
			},
		},
		{
			// Test for Common Table Expression.
			statement:  `with t1 as (select * from t) select * from t1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for UNION.
			statement:  `select 1 as c1, 2 as c2, 3 as c3, 4 UNION ALL select * from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "c1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "c2",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c3",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "?column?",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for UNION.
			statement:  `select * from t UNION ALL select * from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for explicit schema name.
			statement:  `select concat(public.t.a, public.t.b, public.t.c) from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "concat",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
			},
		},
		{
			// Test for associated sub-query.
			statement:  `select a, (select max(b) > y.a from t as x) from t as y`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "?column?",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
			},
		},
		{
			// Test for JOIN with ON clause.
			statement:  `select * from t as t1 join t as t2 on t1.a = t2.a`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for natural JOIN.
			statement:  `select * from t as t1 natural join t as t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for JOIN with USING clause.
			statement:  `select * from t as t1 join t as t2 using(a)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for non-associated sub-query
			statement:  "select t.a, (select max(a) from t) from t as t1 join t on t.a = t1.b",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "max",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
			},
		},
		{
			// Test for functions.
			statement:  `select max(a), a-b as c1, a=b as c2, a>b, b in (a, c, d) from (select * from t) result`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "max",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "c1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "c2",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "?column?",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "?column?",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
			},
		},
		{
			// Test for sub-query
			statement:  "select * from (select * from t) result LIMIT 100000;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for sub-select.
			statement:  "select * from (select a, t.b, public.t.c, d as d1 from public.t) result LIMIT 100000;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for field name.
			statement:  "select a, t.b, public.t.c, d as d1 from t",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d1",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for *.
			statement:  "select * from t",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []base.SensitiveField{
				{
					Name:              "a",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_FULL),
				},
				{
					Name:              "b",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "c",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE),
				},
				{
					Name:              "d",
					MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_PARTIAL),
				},
			},
		},
		{
			// Test for no FROM clause.
			statement:  "select 1;",
			schemaInfo: &base.SensitiveSchemaInfo{},
			fieldList:  []base.SensitiveField{{Name: "?column?", MaskingAttributes: base.NewMaskingAttributes(storepb.MaskingLevel_NONE)}},
		},
		{
			// Test for EXPLAIN statements.
			statement:  "explain select 1;",
			schemaInfo: &base.SensitiveSchemaInfo{},
			fieldList:  nil,
		},
	}

	for _, test := range tests {
		res, err := GetMaskedFields(test.statement, defaultDatabase, test.schemaInfo)
		require.NoError(t, err)
		require.NoError(t, err, test.statement)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}
