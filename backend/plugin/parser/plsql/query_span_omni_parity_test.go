package plsql

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestOracleOmniQuerySpanGoldenHarness is the strict cutover guard. It compares
// the package-internal omni path against the existing YAML corpus instead of
// comparing GetQuerySpan to itself after the production cutover.
func TestOracleOmniQuerySpanGoldenHarness(t *testing.T) {
	type testCase struct {
		Description           string              `yaml:"description,omitempty"`
		Statement             string              `yaml:"statement,omitempty"`
		DefaultDatabase       string              `yaml:"defaultDatabase,omitempty"`
		Metadata              string              `yaml:"metadata,omitempty"`
		CrossDatabaseMetadata string              `yaml:"crossDatabaseMetadata,omitempty"`
		QuerySpan             *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	type diff struct {
		index       int
		description string
		statement   string
		details     string
	}

	const testDataPath = "test-data/query_span.yaml"
	yamlFile, err := os.Open(testDataPath)
	require.NoError(t, err)
	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	require.NoError(t, yamlFile.Close())

	var testCases []testCase
	require.NoError(t, yaml.Unmarshal(byteValue, &testCases))

	var diffs []diff
	for i, tc := range testCases {
		if strings.TrimSpace(tc.Statement) == "" {
			continue
		}

		omni, err := runOracleOmniGoldenCase(context.Background(), tc.Statement, tc.Metadata, tc.CrossDatabaseMetadata, tc.DefaultDatabase)
		if err != nil {
			diffs = append(diffs, diff{
				index:       i,
				description: tc.Description,
				statement:   tc.Statement,
				details:     err.Error(),
			})
			continue
		}
		if !reflect.DeepEqual(tc.QuerySpan, omni) {
			diffs = append(diffs, diff{
				index:       i,
				description: tc.Description,
				statement:   tc.Statement,
				details:     fmt.Sprintf("want=%+v omni=%+v", tc.QuerySpan, omni),
			})
		}
	}

	t.Logf("Oracle omni query-span golden: %d/%d matched, %d diffs", len(testCases)-len(diffs), len(testCases), len(diffs))
	for i, d := range diffs {
		if i >= 10 {
			t.Logf("... %d more diffs omitted", len(diffs)-i)
			break
		}
		t.Logf("[case %d %q] %s\n  SQL: %s", d.index, d.description, d.details, firstOracleOmniProbeLine(d.statement))
	}
	require.Empty(t, diffs)
}

func runOracleOmniGoldenCase(
	ctx context.Context,
	statement string,
	metadataText string,
	crossDatabaseMetadataText string,
	defaultDatabase string,
) (*base.YamlQuerySpan, error) {
	metadata := &storepb.DatabaseSchemaMetadata{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(metadataText), metadata); err != nil {
		return nil, err
	}
	list := []*storepb.DatabaseSchemaMetadata{metadata}
	if crossDatabaseMetadataText != "" {
		crossDatabase := &storepb.DatabaseSchemaMetadata{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(crossDatabaseMetadataText), crossDatabase); err != nil {
			return nil, err
		}
		list = append(list, crossDatabase)
	}

	databaseMetadataGetter, databaseNamesLister, linkedDatabaseMetadataGetter := buildMockDatabaseMetadataGetter(list)
	gCtx := base.GetQuerySpanContext{
		InstanceID:                    instanceIDA,
		GetDatabaseMetadataFunc:       databaseMetadataGetter,
		ListDatabaseNamesFunc:         databaseNamesLister,
		GetLinkedDatabaseMetadataFunc: linkedDatabaseMetadataGetter,
	}

	omni, err := newOmniQuerySpanExtractor(defaultDatabase, gCtx).getOmniQuerySpan(ctx, statement)
	if err != nil {
		return nil, err
	}
	return omni.ToYaml(), nil
}

func TestOracleOmniLongTailTableSources(t *testing.T) {
	tests := []struct {
		name              string
		statement         string
		wantResult        string
		wantSourceColumns []base.ColumnResource
		wantQuerySources  []base.ColumnResource
	}{
		{
			name:              "json table",
			statement:         "SELECT JT.ID FROM T, JSON_TABLE(T.J, '$' COLUMNS (ID NUMBER PATH '$.id')) JT",
			wantResult:        "ID",
			wantSourceColumns: []base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "J"}},
			wantQuerySources:  []base.ColumnResource{{Database: "PUBLIC", Table: "T"}},
		},
		{
			name:              "xml table",
			statement:         "SELECT XT.ID FROM T, XMLTABLE('/root' PASSING T.X COLUMNS ID NUMBER PATH 'id') XT",
			wantResult:        "ID",
			wantSourceColumns: []base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "X"}},
			wantQuerySources:  []base.ColumnResource{{Database: "PUBLIC", Table: "T"}},
		},
		{
			name:              "containers",
			statement:         "SELECT A FROM CONTAINERS(T)",
			wantResult:        "A",
			wantSourceColumns: []base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}},
			wantQuerySources:  []base.ColumnResource{{Database: "PUBLIC", Table: "T"}},
		},
		{
			name:              "inline external table",
			statement:         "SELECT EXT.A FROM EXTERNAL ((A NUMBER, B VARCHAR2(20)) TYPE ORACLE_LOADER DEFAULT DIRECTORY D ACCESS PARAMETERS (FIELDS TERMINATED BY ',') LOCATION ('x.csv')) EXT",
			wantResult:        "A",
			wantSourceColumns: []base.ColumnResource{},
			wantQuerySources:  []base.ColumnResource{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Len(t, span.Results, 1)
			require.Equal(t, test.wantResult, span.Results[0].Name)
			require.Equal(t, sourceColumnSetFromList(test.wantSourceColumns), span.Results[0].SourceColumns)
			require.Equal(t, sourceColumnSetFromList(test.wantQuerySources), span.SourceColumns)
		})
	}
}

func TestOracleOmniAnalyticCountStarSources(t *testing.T) {
	span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(
		context.Background(),
		"SELECT COUNT(*) OVER (PARTITION BY B ORDER BY A) AS C FROM T",
	)
	require.NoError(t, err)
	require.NotNil(t, span)
	require.Len(t, span.Results, 1)
	require.Equal(t, "C", span.Results[0].Name)
	require.Equal(t, sourceColumnSetFromList([]base.ColumnResource{
		{Database: "PUBLIC", Table: "T", Column: "A"},
		{Database: "PUBLIC", Table: "T", Column: "B"},
	}), span.Results[0].SourceColumns)
}

func TestOracleOmniUnqualifiedStarExpandsAllCommaSources(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "base tables",
			statement: "SELECT * FROM T, T2",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "C"}})},
				{Name: "J", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "J"}})},
				{Name: "X", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "X"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "C"}})},
				{Name: "D", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "D"}})},
			},
		},
		{
			name:      "json table",
			statement: "SELECT * FROM T, JSON_TABLE(T.J, '$' COLUMNS (ID NUMBER PATH '$.id')) JT",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "C"}})},
				{Name: "J", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "J"}})},
				{Name: "X", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "X"}})},
				{Name: "ID", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "J"}})},
			},
		},
		{
			name:      "join",
			statement: "SELECT * FROM T JOIN T2 ON T.A = T2.C",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "C"}})},
				{Name: "J", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "J"}})},
				{Name: "X", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "X"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "C"}})},
				{Name: "D", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "D"}})},
			},
		},
		{
			name:      "natural join",
			statement: "SELECT * FROM T NATURAL JOIN T2",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "C"},
					{Database: "PUBLIC", Table: "T2", Column: "C"},
				})},
				{Name: "J", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "J"}})},
				{Name: "X", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "X"}})},
				{Name: "D", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "D"}})},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Len(t, span.Results, len(test.want))
			for i, want := range test.want {
				require.Equal(t, want.Name, span.Results[i].Name)
				require.Equal(t, want.SourceColumns, span.Results[i].SourceColumns)
			}
		})
	}
}

func TestOracleOmniCTEScopeAndSetOperations(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "sibling cte references previous cte",
			statement: "WITH C1 AS (SELECT A FROM T), C2 AS (SELECT A FROM C1 UNION SELECT C FROM T2) SELECT * FROM C2",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "A"},
					{Database: "PUBLIC", Table: "T2", Column: "C"},
				})},
			},
		},
		{
			name:      "cte name shadows physical table",
			statement: "WITH T AS (SELECT C FROM T2) SELECT * FROM T",
			want: []base.QuerySpanResult{
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "C"}})},
			},
		},
		{
			name:      "nested cte name shadows outer cte",
			statement: "WITH C AS (WITH C AS (SELECT C FROM T2) SELECT C FROM C) SELECT * FROM C",
			want: []base.QuerySpanResult{
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "C"}})},
			},
		},
		{
			name:      "multiple ctes join",
			statement: "WITH C1 AS (SELECT A FROM T), C2 AS (SELECT C FROM T2) SELECT C1.A, C2.C FROM C1 JOIN C2 ON C1.A = C2.C",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "C"}})},
			},
		},
		{
			name:      "intersect merges source columns positionally",
			statement: "SELECT A FROM T INTERSECT SELECT C FROM T2",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "A"},
					{Database: "PUBLIC", Table: "T2", Column: "C"},
				})},
			},
		},
		{
			name:      "minus merges source columns positionally",
			statement: "SELECT A FROM T MINUS SELECT C FROM T2",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "A"},
					{Database: "PUBLIC", Table: "T2", Column: "C"},
				})},
			},
		},
		{
			name:      "recursive cte reaches stable source closure",
			statement: "WITH C(X) AS (SELECT A FROM T UNION ALL SELECT T2.C FROM T2 JOIN C ON C.X = T2.C) SELECT * FROM C",
			want: []base.QuerySpanResult{
				{Name: "X", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "A"},
					{Database: "PUBLIC", Table: "T2", Column: "C"},
				})},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Equal(t, test.want, span.Results)
		})
	}
}

func TestOracleOmniExpressionAndAccessTableCoverage(t *testing.T) {
	t.Run("expression nodes keep source columns", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(
			context.Background(),
			`SELECT
  CASE WHEN A > 0 THEN B ELSE C END AS CASE_VALUE,
  DECODE(A, 1, B, C) AS DECODE_VALUE,
  CAST(A AS NUMBER) AS CAST_VALUE,
  B BETWEEN A AND C AS BETWEEN_VALUE,
  X LIKE J AS LIKE_VALUE
FROM T`,
		)
		require.NoError(t, err)
		require.NotNil(t, span)
		want := []base.QuerySpanResult{
			{Name: "CASE_VALUE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
				{Database: "PUBLIC", Table: "T", Column: "A"},
				{Database: "PUBLIC", Table: "T", Column: "B"},
				{Database: "PUBLIC", Table: "T", Column: "C"},
			})},
			{Name: "DECODE_VALUE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
				{Database: "PUBLIC", Table: "T", Column: "A"},
				{Database: "PUBLIC", Table: "T", Column: "B"},
				{Database: "PUBLIC", Table: "T", Column: "C"},
			})},
			{Name: "CAST_VALUE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
			{Name: "BETWEEN_VALUE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
				{Database: "PUBLIC", Table: "T", Column: "A"},
				{Database: "PUBLIC", Table: "T", Column: "B"},
				{Database: "PUBLIC", Table: "T", Column: "C"},
			})},
			{Name: "LIKE_VALUE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
				{Database: "PUBLIC", Table: "T", Column: "J"},
				{Database: "PUBLIC", Table: "T", Column: "X"},
			})},
		}
		require.Equal(t, want, span.Results)
	})

	t.Run("subqueries in non result clauses contribute access tables", func(t *testing.T) {
		tests := []string{
			"SELECT A FROM T WHERE EXISTS (SELECT C FROM T2 WHERE T2.C = T.A)",
			"SELECT A FROM T ORDER BY (SELECT C FROM T2)",
			"SELECT COALESCE((SELECT C FROM T2), A) FROM T",
		}
		for _, statement := range tests {
			span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			require.NotNil(t, span, statement)
			require.Equal(t, sourceColumnSetFromList([]base.ColumnResource{
				{Database: "PUBLIC", Table: "T"},
				{Database: "PUBLIC", Table: "T2"},
			}), span.SourceColumns, statement)
		}
	})

	t.Run("cursor expression uses subquery scope", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(
			context.Background(),
			"SELECT CURSOR(SELECT C FROM T2) AS CUR FROM T",
		)
		require.NoError(t, err)
		require.NotNil(t, span)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "CUR", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T2", Column: "C"}})},
		}, span.Results)
	})
}

func TestOracleOmniResourceNotFoundPropagation(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "top level table",
			statement: "SELECT A FROM MISSING_TABLE",
		},
		{
			name:      "predicate subquery",
			statement: "SELECT A FROM T WHERE EXISTS (SELECT 1 FROM MISSING_TABLE)",
		},
		{
			name:      "target subquery",
			statement: "SELECT (SELECT A FROM MISSING_TABLE) AS A FROM T",
		},
		{
			name:      "recursive cte arm",
			statement: "WITH C(X) AS (SELECT A FROM T UNION ALL SELECT MISSING_TABLE.A FROM MISSING_TABLE JOIN C ON C.X = MISSING_TABLE.A) SELECT * FROM C",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Empty(t, span.Results)
			require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))
		})
	}
}

func TestOracleOmniTypedLongTailTableSources(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "table collection column aliases",
			statement: "SELECT * FROM TABLE(pkg.values()) X(A, B)",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList(nil)},
				{Name: "B", SourceColumns: sourceColumnSetFromList(nil)},
			},
		},
		{
			name:      "pivot typed output",
			statement: "SELECT * FROM (SELECT A, B FROM T) PIVOT (COUNT(*) AS CNT FOR B IN (1 AS ONE, 2 AS TWO))",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "ONE_CNT", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
				{Name: "TWO_CNT", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
			},
		},
		{
			name:      "pivot explicit projection",
			statement: "SELECT A FROM (SELECT A, B FROM T) PIVOT (COUNT(*) AS CNT FOR B IN (1 AS ONE, 2 AS TWO))",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
			},
		},
		{
			name:      "pivot aggregate input is consumed",
			statement: "SELECT * FROM (SELECT A, B, C FROM T) PIVOT (SUM(C) AS S FOR B IN (1 AS ONE))",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "ONE_S", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "B"},
					{Database: "PUBLIC", Table: "T", Column: "C"},
				})},
			},
		},
		{
			name:      "pivot aggregate projected alias input is consumed",
			statement: "SELECT * FROM (SELECT A, B, C AS D FROM T) PIVOT (SUM(D) AS S FOR B IN (1 AS ONE))",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "ONE_S", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "B"},
					{Database: "PUBLIC", Table: "T", Column: "C"},
				})},
			},
		},
		{
			name:      "unpivot typed mappings",
			statement: "SELECT * FROM (SELECT A, B, C FROM T) UNPIVOT (VAL FOR COL IN (B AS 'B', C AS 'C'))",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "VAL", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "B"},
					{Database: "PUBLIC", Table: "T", Column: "C"},
				})},
				{Name: "COL", SourceColumns: sourceColumnSetFromList(nil)},
			},
		},
		{
			name:      "unpivot explicit projection",
			statement: "SELECT VAL FROM (SELECT A, B, C FROM T) UNPIVOT (VAL FOR COL IN (B AS 'B', C AS 'C'))",
			want: []base.QuerySpanResult{
				{Name: "VAL", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{
					{Database: "PUBLIC", Table: "T", Column: "B"},
					{Database: "PUBLIC", Table: "T", Column: "C"},
				})},
			},
		},
		{
			name:      "model dimension and measures",
			statement: "SELECT * FROM T MODEL DIMENSION BY (A) MEASURES (B) RULES (B[1] = 1)",
			want: []base.QuerySpanResult{
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
			},
		},
		{
			name:      "model explicit projection",
			statement: "SELECT B FROM T MODEL DIMENSION BY (A) MEASURES (B) RULES (B[1] = 1)",
			want: []base.QuerySpanResult{
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
			},
		},
		{
			name:      "model partition dimension and measures",
			statement: "SELECT * FROM T MODEL PARTITION BY (C) DIMENSION BY (A) MEASURES (B) RULES (B[1] = 1)",
			want: []base.QuerySpanResult{
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "C"}})},
				{Name: "A", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "A"}})},
				{Name: "B", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "B"}})},
			},
		},
		{
			name:      "model partition explicit projection",
			statement: "SELECT C FROM T MODEL PARTITION BY (C) DIMENSION BY (A) MEASURES (B) RULES (B[1] = 1)",
			want: []base.QuerySpanResult{
				{Name: "C", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "T", Column: "C"}})},
			},
		},
		{
			name: "match recognize measures",
			statement: `SELECT * FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(PRICE) AS FIRST_PRICE, LAST(PRICE) AS LAST_PRICE
  ONE ROW PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`,
			want: []base.QuerySpanResult{
				{Name: "ACCOUNT_ID", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "ACCOUNT_ID"}})},
				{Name: "FIRST_PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
				{Name: "LAST_PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
			},
		},
		{
			name: "match recognize partition projection",
			statement: `SELECT ACCOUNT_ID FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(PRICE) AS FIRST_PRICE
  ONE ROW PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`,
			want: []base.QuerySpanResult{
				{Name: "ACCOUNT_ID", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "ACCOUNT_ID"}})},
			},
		},
		{
			name: "match recognize pattern variable measures",
			statement: `SELECT * FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(A.PRICE) AS FIRST_PRICE, LAST(B.PRICE) AS LAST_PRICE
  ONE ROW PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`,
			want: []base.QuerySpanResult{
				{Name: "ACCOUNT_ID", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "ACCOUNT_ID"}})},
				{Name: "FIRST_PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
				{Name: "LAST_PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
			},
		},
		{
			name: "match recognize all rows keeps input columns",
			statement: `SELECT * FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(PRICE) AS FIRST_PRICE
  ALL ROWS PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`,
			want: []base.QuerySpanResult{
				{Name: "ACCOUNT_ID", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "ACCOUNT_ID"}}), IsPlainField: true},
				{Name: "TRADE_TIME", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "TRADE_TIME"}}), IsPlainField: true},
				{Name: "PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}}), IsPlainField: true},
				{Name: "FIRST_PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
			},
		},
		{
			name: "match recognize all rows pattern variable measure",
			statement: `SELECT * FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(A.PRICE) AS FIRST_PRICE
  ALL ROWS PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`,
			want: []base.QuerySpanResult{
				{Name: "ACCOUNT_ID", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "ACCOUNT_ID"}}), IsPlainField: true},
				{Name: "TRADE_TIME", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "TRADE_TIME"}}), IsPlainField: true},
				{Name: "PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}}), IsPlainField: true},
				{Name: "FIRST_PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
			},
		},
		{
			name: "match recognize all rows input projection",
			statement: `SELECT PRICE FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(PRICE) AS FIRST_PRICE
  ALL ROWS PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`,
			want: []base.QuerySpanResult{
				{Name: "PRICE", SourceColumns: sourceColumnSetFromList([]base.ColumnResource{{Database: "PUBLIC", Table: "TRADES", Column: "PRICE"}})},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("PUBLIC", oracleOmniLongTailTestContext(t)).getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Equal(t, test.want, span.Results)
		})
	}
}

func oracleOmniLongTailTestContext(t *testing.T) base.GetQuerySpanContext {
	t.Helper()

	const metadataText = `{
		"name": "PUBLIC",
		"schemas": [{
			"name": "",
			"tables": [{
			"name": "T",
			"columns": [
				{"name": "A"},
				{"name": "B"},
				{"name": "C"},
				{"name": "J"},
				{"name": "X"}
			]
		}, {
			"name": "T2",
			"columns": [
				{"name": "C"},
				{"name": "D"}
			]
		}, {
			"name": "TRADES",
			"columns": [
				{"name": "ACCOUNT_ID"},
				{"name": "TRADE_TIME"},
				{"name": "PRICE"}
			]
		}]
	}]
}`

	metadata := &storepb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(metadataText), metadata))
	databaseMetadataGetter, databaseNamesLister, linkedDatabaseMetadataGetter := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	return base.GetQuerySpanContext{
		InstanceID:                    instanceIDA,
		GetDatabaseMetadataFunc:       databaseMetadataGetter,
		ListDatabaseNamesFunc:         databaseNamesLister,
		GetLinkedDatabaseMetadataFunc: linkedDatabaseMetadataGetter,
	}
}

func firstOracleOmniProbeLine(statement string) string {
	for _, line := range strings.Split(statement, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func sourceColumnSetFromList(columns []base.ColumnResource) base.SourceColumnSet {
	result := make(base.SourceColumnSet, len(columns))
	for _, column := range columns {
		result[column] = true
	}
	return result
}
