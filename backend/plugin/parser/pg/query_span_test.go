package pg

import (
	"context"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpan(t *testing.T) {
	type testCase struct {
		Description     string `yaml:"description,omitempty"`
		Statement       string `yaml:"statement,omitempty"`
		DefaultDatabase string `yaml:"defaultDatabase,omitempty"`
		// Metadata is the protojson encoded metadatapb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string              `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	const (
		record = false
	)

	var (
		testDataPaths = []string{"test-data/query_span.yaml", "test-data/query_type.yaml"}
	)

	a := require.New(t)
	for _, testDataPath := range testDataPaths {
		yamlFile, err := os.Open(testDataPath)
		a.NoError(err)

		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))

		for i, tc := range testCases {
			metadata := &metadatapb.DatabaseSchemaMetadata{}
			a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata))
			databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: databaseMetadataGetter,
				ListDatabaseNamesFunc:   databaseNameLister,
			}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "", false)
			a.NoErrorf(err, "idx: %d statement: %s", i, tc.Statement)
			resultYaml := result.ToYaml()
			if record {
				testCases[i].QuerySpan = resultYaml
			} else {
				a.Equal(tc.QuerySpan, resultYaml, "idx: %d statement: %s", i, tc.Statement)
			}
		}

		if record {
			yamltest.Record(t, testDataPath, testCases)
		}
	}
}

// TestGetQuerySpanNilMetadata ensures a never-synced database (nil metadata)
// returns an error instead of panicking. Access Grant approval evaluation
// calls GetQuerySpan for databases that may not have been schema-synced yet.
func TestGetQuerySpanNilMetadata(t *testing.T) {
	_, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
	}, base.Statement{Text: "SELECT 1;"}, "unsynced-db", "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "database metadata for database \"unsynced-db\" not found")
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return databaseName, databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}

func TestQuerySpanSurvivesABadQuotedIdentifier(t *testing.T) {
	// A table name carrying a character sequence that a deparse-and-reparse
	// loop chokes on (BYT-9215). Loading the snapshot never renders DDL, so the
	// table installs and unrelated queries succeed either way.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				// A quoted identifier containing an apostrophe — the kind of
				// input BYT-9215 reported failing under Exec(ddl).
				{
					Name: "'weird'table",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int4"},
					},
				},
				// An unrelated healthy table.
				{
					Name: "accounts",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "email", Type: "text"},
					},
				},
			},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, email FROM accounts`)
	if len(span.Results) != 2 {
		t.Fatalf("accounts query: got %d results, want 2", len(span.Results))
	}
	for _, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result %q has empty sources (lineage lost)", r.Name)
		}
	}
}

func TestGetQuerySpanWithSelectedSchemaFallsBackToPublic(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "app",
			},
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name: "customer",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "ssn", Type: "text"},
					},
				}},
			},
		},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{meta})
	span, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: `SELECT ssn FROM customer`}, "db", "app", false)
	if err != nil {
		t.Fatalf("GetQuerySpan: %v", err)
	}
	if len(span.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(span.Results))
	}
	want := base.ColumnResource{Database: "db", Schema: "public", Table: "customer", Column: "ssn"}
	if _, ok := span.Results[0].SourceColumns[want]; !ok {
		t.Fatalf("result %q missing public fallback source %+v; have %+v", span.Results[0].Name, want, span.Results[0].SourceColumns)
	}
	wantAccess := base.ColumnResource{Database: "db", Schema: "public", Table: "customer"}
	if _, ok := span.SourceColumns[wantAccess]; !ok {
		t.Fatalf("span missing public fallback access source %+v; have %+v", wantAccess, span.SourceColumns)
	}
}

func TestGetQuerySpanSelectedSchemaTakesPrecedenceOverPublic(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "app",
				Tables: []*metadatapb.TableMetadata{{
					Name: "customer",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "email", Type: "text"},
					},
				}},
			},
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name: "customer",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "ssn", Type: "text"},
					},
				}},
			},
		},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{meta})
	span, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: `SELECT email FROM customer`}, "db", "app", false)
	if err != nil {
		t.Fatalf("GetQuerySpan: %v", err)
	}
	if len(span.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(span.Results))
	}
	want := base.ColumnResource{Database: "db", Schema: "app", Table: "customer", Column: "email"}
	if _, ok := span.Results[0].SourceColumns[want]; !ok {
		t.Fatalf("result %q missing selected-schema source %+v; have %+v", span.Results[0].Name, want, span.Results[0].SourceColumns)
	}
}

func TestQuerySpanSurvivesABrokenEnum(t *testing.T) {
	// The table references a user-defined type the snapshot never declares, so
	// it installs as a stand-in with text columns. A query against it still
	// returns the column names the snapshot records.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "tasks",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					// References an enum that is NOT declared in metadata —
					// buildCreateStmt will succeed (typeNameFromString works)
					// but DefineRelation fails (enum not in catalog).
					{Name: "status", Type: "public.nonexistent_enum"},
					{Name: "title", Type: "text"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, status, title FROM tasks`)
	if len(span.Results) != 3 {
		t.Fatalf("tasks query: got %d results, want 3", len(span.Results))
	}
	wantNames := []string{"id", "status", "title"}
	for i, want := range wantNames {
		if i >= len(span.Results) {
			break
		}
		if span.Results[i].Name != want {
			t.Errorf("result[%d]: got %q, want %q", i, span.Results[i].Name, want)
		}
		if len(span.Results[i].SourceColumns) == 0 {
			t.Errorf("result[%d] (%s): empty sources", i, span.Results[i].Name)
		}
	}
}

func TestQuerySpanKeepsAHealthyNeighborOfABrokenTable(t *testing.T) {
	// A query against a healthy table must succeed even when an unrelated
	// table in the same schema references a broken type. This is the core
	// blast-radius claim: one bad object does not poison all queries.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				{
					Name: "broken",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "bad", Type: "public.nonexistent_type"},
					},
				},
				{
					Name: "healthy",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "int4"},
						{Name: "label", Type: "text"},
					},
				},
			},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, label FROM healthy`)
	if len(span.Results) != 2 {
		t.Fatalf("healthy query: got %d results, want 2", len(span.Results))
	}
	for _, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result %q lost lineage because of unrelated broken table", r.Name)
		}
	}
}

func TestQuerySpanResolvesAViewOverABrokenTable(t *testing.T) {
	// Chain: enum missing → table T references it, so T stands in → view V over
	// T installs against that stand-in → a query on V still resolves.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "records",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.nonexistent_enum"},
					{Name: "title", Type: "text"},
				},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "records_view",
				Definition: "SELECT id, title, status FROM records",
				DependencyColumns: []*metadatapb.DependencyColumn{
					{Schema: "public", Table: "records", Column: "id"},
					{Schema: "public", Table: "records", Column: "title"},
					{Schema: "public", Table: "records", Column: "status"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, title, status FROM records_view`)
	if len(span.Results) != 3 {
		t.Fatalf("view query: got %d results, want 3", len(span.Results))
	}
	for i, r := range span.Results {
		if len(r.SourceColumns) == 0 {
			t.Errorf("result[%d] (%s) lost lineage", i, r.Name)
		}
	}
}

func TestQuerySpanOnAHealthySchema(t *testing.T) {
	// Baseline: a clean schema produces exact lineage down to
	// (schema, table, column).
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "users",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "email", Type: "text"},
					{Name: "created_at", Type: "timestamp with time zone"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, email FROM users WHERE id > 0`)
	if len(span.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "users", "id")
	mustHaveExactSource(t, span.Results[1], "users", "email")
}

func TestQuerySpanKeepsLineageThroughAPartition(t *testing.T) {
	// Masking resolves a partition to its parent table, which needs the read to
	// trace to the partition's columns.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name:       "orders",
				Columns:    []*metadatapb.ColumnMetadata{{Name: "id", Type: "int4"}, {Name: "ssn", Type: "text"}},
				Partitions: []*metadatapb.TablePartitionMetadata{{Name: "orders_2024"}},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT ssn FROM orders_2024`)
	if len(span.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "orders_2024", "ssn")
}

func TestQuerySpanResolvesADeclaredEnum(t *testing.T) {
	// When an enum IS declared in metadata, the table installs as real and
	// enum-typed columns resolve through their real type. This is the
	// positive control for TestQuerySpanSurvivesABrokenEnum.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			EnumTypes: []*metadatapb.EnumTypeMetadata{{
				Name:   "task_status",
				Values: []string{"pending", "running", "done"},
			}},
			Tables: []*metadatapb.TableMetadata{{
				Name: "tasks",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.task_status"},
				},
			}},
		}},
	}

	span := mustGetQuerySpan(t, meta, `SELECT id, status FROM tasks WHERE status = 'pending'`)
	if len(span.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[0], "tasks", "id")
	mustHaveExactSource(t, span.Results[1], "tasks", "status")
}

func TestQuerySpanOnACorrelatedRangeFunctionSubquery(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				{
					Name: "compliance_case_record_audits",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "case_id", Type: "text"},
						{Name: "entity_id", Type: "text"},
						{Name: "reason", Type: "text"},
						{Name: "remark", Type: "text"},
						{Name: "case_record", Type: "jsonb"},
						{Name: "reviewer_by", Type: "text"},
						{Name: "deleted_at", Type: "timestamptz"},
					},
				},
				{
					Name: "compliance_cases",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "case_id", Type: "text"},
						{Name: "case_info", Type: "jsonb"},
						{Name: "deleted_at", Type: "timestamptz"},
					},
				},
			},
		}},
	}
	sql := `
select
  a.case_id,
  a.entity_id as uid,
  a.reason,
  a.remark,
  (
    select elem->>'matchedDateTimeValue'
    from jsonb_array_elements(a.case_record::jsonb->'secondaryFieldResults') elem
    where elem->>'typeId' = '******'
    limit 1
  ) as wc_dob,
  (
    select elem->>'matchedValue'
    from jsonb_array_elements(a.case_record::jsonb->'secondaryFieldResults') elem
    where elem->>'typeId' = '******'
    limit 1
  ) as wc_citizenship,
  c.case_info->>'birth_date' as kyc_dob,
  c.case_info->>'nationality' as kyc_citizenship,
  a.reviewer_by as review_by
from compliance_case_record_audits a
inner join compliance_cases c
  on c.case_id = a.case_id and c.deleted_at is null
where a.reviewer_by in ('****** ', '******')
  and a.deleted_at is null;
`
	span := mustGetQuerySpan(t, meta, sql)
	if len(span.Results) != 9 {
		t.Fatalf("got %d results, want 9", len(span.Results))
	}
	mustHaveExactSource(t, span.Results[4], "compliance_case_record_audits", "case_record")
	mustHaveExactSource(t, span.Results[5], "compliance_case_record_audits", "case_record")
	mustHaveExactSource(t, span.Results[6], "compliance_cases", "case_info")
	mustHaveExactSource(t, span.Results[7], "compliance_cases", "case_info")
}

func TestQuerySpanKeepsJSONBLineageThroughCTEsAndALateralJoin(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "ai_conversation",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "text"},
					{Name: "parts", Type: "jsonb"},
				},
			}},
		}},
	}

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "single_cte",
			sql: `
WITH convs AS (
  SELECT ac.id, ac.parts AS conv_parts
  FROM ai_conversation ac
  LIMIT 1
)
SELECT
  LEFT(
    COALESCE(
      (SELECT string_agg(v::text, ' ')
       FROM jsonb_array_elements_text(
         jsonb_path_query_array(t.part->'bodyData', 'strict $.**.text', '{}'::jsonb, true)
       ) AS v),
      '<no text>'
    ),
    2000
  ) AS content
FROM convs c
CROSS JOIN LATERAL jsonb_array_elements(c.conv_parts) WITH ORDINALITY AS t(part, part_ord)
WHERE t.part->>'type' IN ('prompt', 'text');
`,
		},
		{
			name: "multiple_ctes",
			sql: `
WITH cte1 AS (
  SELECT ac.id, ac.parts AS conv_parts
  FROM ai_conversation ac
  LIMIT 1
),
cte2 AS (
  SELECT id, conv_parts
  FROM cte1
)
SELECT
  LEFT(
    COALESCE(
      (SELECT string_agg(v::text, ' ')
       FROM jsonb_array_elements_text(
         jsonb_path_query_array(t.part->'bodyData', 'strict $.**.text', '{}'::jsonb, true)
       ) AS v),
      '<no text>'
    ),
    2000
  ) AS content
FROM cte2 c
CROSS JOIN LATERAL jsonb_array_elements(c.conv_parts) WITH ORDINALITY AS t(part, part_ord)
WHERE t.part->>'type' IN ('prompt', 'text');
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := mustGetQuerySpan(t, meta, tt.sql)
			if len(span.Results) != 1 {
				t.Fatalf("got %d results, want 1", len(span.Results))
			}
			mustHaveExactSource(t, span.Results[0], "ai_conversation", "parts")
		})
	}
}

func TestAppendQueryDoesNotMutateSharedStack(t *testing.T) {
	outer := &catalog.Query{}
	parent := &catalog.Query{}
	current := &catalog.Query{}
	nested := &catalog.Query{}
	queryStack := []*catalog.Query{outer, parent, current}

	nextStack := appendQuery(queryStack[:2], nested)

	if queryStack[2] != current {
		t.Fatalf("appendQuery mutated caller stack: got %p, want %p", queryStack[2], current)
	}
	if len(nextStack) != 3 || nextStack[2] != nested {
		t.Fatalf("appendQuery returned unexpected stack: %+v", nextStack)
	}
}

func TestWalkExprStopsOnSelfReferentialFunction(t *testing.T) {
	mustNotOverflow(t, func() {
		varExpr := &catalog.VarExpr{RangeIdx: 0, AttNum: 1}
		q := &catalog.Query{
			TargetList: []*catalog.TargetEntry{{
				Expr: varExpr,
			}},
			RangeTable: []*catalog.RangeTableEntry{{
				Kind: catalog.RTEFunction,
				FuncExprs: []catalog.AnalyzedExpr{&catalog.FuncCallExpr{
					Args: []catalog.AnalyzedExpr{&catalog.OpExpr{
						Left: varExpr,
						Right: &catalog.ConstExpr{
							Value: "items",
						},
					}},
				}},
			}},
		}
		extractor := newOmniQuerySpanExtractor(testDB, []string{testSchema}, base.GetQuerySpanContext{})
		extractor.cat = catalog.New()
		extractor.walkExpr(q, q.TargetList[0].Expr, make(base.SourceColumnSet))
	})
}

func TestIsUltimatelyPlainColumnStopsOnCyclicSubquery(t *testing.T) {
	mustNotOverflow(t, func() {
		varExpr := &catalog.VarExpr{RangeIdx: 0, AttNum: 1}
		q := &catalog.Query{
			TargetList: []*catalog.TargetEntry{{
				Expr: varExpr,
			}},
			RangeTable: []*catalog.RangeTableEntry{{
				Kind: catalog.RTESubquery,
			}},
		}
		q.RangeTable[0].Subquery = q
		_ = isUltimatelyPlainColumn(q, varExpr)
	})
}

func TestCollectQueryPredicateColumnsStopsOnCyclicQuery(t *testing.T) {
	mustNotOverflow(t, func() {
		q := &catalog.Query{}
		q.CTEList = []*catalog.CommonTableExprQ{{Query: q}}
		extractor := newOmniQuerySpanExtractor(testDB, []string{testSchema}, base.GetQuerySpanContext{})
		analyzer := &plpgsqlAnalyzer{
			extractor: extractor,
			scope:     newVariableScope(nil),
		}
		analyzer.collectQueryPredicateColumns(q)
	})
}

func TestExtractLineageStopsOnCyclicSetOp(t *testing.T) {
	mustNotOverflow(t, func() {
		q := &catalog.Query{SetOp: catalog.SetOpUnion}
		q.LArg = q
		q.RArg = &catalog.Query{TargetList: []*catalog.TargetEntry{{
			Expr: &catalog.ConstExpr{
				Value: "1",
			},
		}}}
		selStmt := &ast.SelectStmt{}
		selStmt.Larg = selStmt
		selStmt.Rarg = &ast.SelectStmt{}
		extractor := newOmniQuerySpanExtractor(testDB, []string{testSchema}, base.GetQuerySpanContext{})
		extractor.extractLineage(q, selStmt)
	})
}

func TestResolveThroughCTEStopsOnCyclicCTE(t *testing.T) {
	mustNotOverflow(t, func() {
		cteSel := &ast.SelectStmt{
			TargetList: &ast.List{Items: []ast.Node{&ast.ResTarget{
				Name: "x",
				Val: &ast.ColumnRef{Fields: &ast.List{Items: []ast.Node{
					&ast.String{Str: "c"},
					&ast.String{Str: "x"},
				}}},
			}}},
			FromClause: &ast.List{Items: []ast.Node{&ast.RangeVar{Relname: "c"}}},
		}
		extractor := newOmniQuerySpanExtractor(testDB, []string{testSchema}, base.GetQuerySpanContext{})
		extractor.cat = catalog.New()
		analyzer := &plpgsqlAnalyzer{
			extractor: extractor,
			scope:     newVariableScope(nil),
			cteMap:    map[string]*ast.SelectStmt{"c": cteSel},
		}
		analyzer.resolveThroughCTE(cteSel, "x", nil, make(base.SourceColumnSet))
	})
}

// ---------- helpers ----------

// testDB / testSchema are the fixed database and schema names these tests use;
// as constants they keep the call sites terse and satisfy the unparam linter.
const (
	testDB     = "db"
	testSchema = "public"
)

// noOverflowEnv marks the subprocess mustNotOverflow spawns.
const noOverflowEnv = "BYTEBASE_TEST_NO_OVERFLOW"

// mustNotOverflow runs fn in a subprocess with a 1 MB stack, so recursion that
// does not terminate fails this test instead of taking the whole run down. The
// subprocess re-runs this same test, which then takes the fn branch.
func mustNotOverflow(t *testing.T, fn func()) {
	t.Helper()
	if os.Getenv(noOverflowEnv) == "1" {
		debug.SetMaxStack(1 << 20)
		fn()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
	cmd.Env = append(os.Environ(), noOverflowEnv+"=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("recursion did not terminate: %v\n%s", err, output)
	}
}

func mustGetQuerySpan(t *testing.T, meta *metadatapb.DatabaseSchemaMetadata, sql string) *base.QuerySpan {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{meta})
	span, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: sql}, testDB, "", false)
	if err != nil {
		t.Fatalf("GetQuerySpan(%q): %v", sql, err)
	}
	if span == nil {
		t.Fatalf("GetQuerySpan(%q): nil span", sql)
	}
	return span
}

func mustHaveExactSource(t *testing.T, result base.QuerySpanResult, table, column string) {
	t.Helper()
	want := base.ColumnResource{
		Database: testDB,
		Schema:   testSchema,
		Table:    table,
		Column:   column,
	}
	for src := range result.SourceColumns {
		if src == want {
			return
		}
	}
	t.Errorf("result %q missing source %+v; have %+v", result.Name, want, result.SourceColumns)
}

// TestRangeVarFallbackSkipsCompositeTypes proves the query-span fallback does
// not expand a standalone composite type as a FROM source — PostgreSQL
// rejects composite types as table sources.
func TestRangeVarFallbackSkipsCompositeTypes(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "t",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
			},
		},
	}

	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	if _, err := cat.LoadMetadata(context.Background(), meta, catalog.LoadMetadataOptions{}); err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	e := &omniQuerySpanExtractor{
		cat:             cat,
		searchPath:      []string{"public"},
		defaultDatabase: "db",
	}
	if results := e.extractColumnsFromRangeVar(&ast.RangeVar{Schemaname: "public", Relname: "addr"}); results != nil {
		t.Errorf("composite type must not expand as a FROM source, got %d columns", len(results))
	}
	if results := e.extractColumnsFromRangeVar(&ast.RangeVar{Schemaname: "public", Relname: "t"}); len(results) != 1 {
		t.Errorf("real table must still expand, got %d columns", len(results))
	}
}
