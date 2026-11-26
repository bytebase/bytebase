package v1

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	elasticsearchparser "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
)

// testElasticsearchObjectSchema creates a schema where "email" and "contact.phone" are masked.
func testElasticsearchObjectSchema() *storepb.ObjectSchema {
	return &storepb.ObjectSchema{
		Kind: &storepb.ObjectSchema_StructKind_{
			StructKind: &storepb.ObjectSchema_StructKind{
				Properties: map[string]*storepb.ObjectSchema{
					"name":  {Type: storepb.ObjectSchema_STRING},
					"email": {Type: storepb.ObjectSchema_STRING, SemanticType: masker.DefaultSemanticTypeID},
					"age":   {Type: storepb.ObjectSchema_NUMBER},
					"contact": {
						Type: storepb.ObjectSchema_OBJECT,
						Kind: &storepb.ObjectSchema_StructKind_{
							StructKind: &storepb.ObjectSchema_StructKind{
								Properties: map[string]*storepb.ObjectSchema{
									"phone": {Type: storepb.ObjectSchema_STRING, SemanticType: masker.DefaultSemanticTypeID},
									"city":  {Type: storepb.ObjectSchema_STRING},
								},
							},
						},
					},
				},
			},
		},
	}
}

// anyIndex masks a hit of any index by the schema given.
func anyIndex(schema *storepb.ObjectSchema) elasticsearchIndexLookup {
	return func(string) (*storepb.ObjectSchema, bool) { return schema, true }
}

func testElasticsearchMaskerMap() map[string]masker.Masker {
	return map[string]masker.Masker{
		masker.DefaultSemanticTypeID: masker.NewDefaultFullMasker(),
	}
}

type elasticsearchTestData struct {
	LookupSemanticType   []lookupSemanticTypeCase `yaml:"lookupSemanticType"`
	MaskHitsColumn       []maskJSONCase           `yaml:"maskHitsColumn"`
	MaskDocSource        []maskJSONCase           `yaml:"maskDocSource"`
	MaskGetSourceColumn  []maskGetSourceCase      `yaml:"maskGetSourceColumn"`
	MaskMGetSource       []maskJSONCase           `yaml:"maskMGetSource"`
	MaskMSearchResponses []maskJSONCase           `yaml:"maskMSearchResponses"`
	MaskInnerHits        []maskJSONCase           `yaml:"maskInnerHits"`
	CheckBlocked         []checkBlockedCase       `yaml:"checkBlocked"`
}

type lookupSemanticTypeCase struct {
	Description string `yaml:"description"`
	DotPath     string `yaml:"dotPath"`
	WantType    string `yaml:"wantType"`
}

type maskJSONCase struct {
	Description string   `yaml:"description"`
	Input       string   `yaml:"input"`
	Want        string   `yaml:"want"`
	SortFields  []string `yaml:"sortFields"`
}

type maskGetSourceCase struct {
	Description string `yaml:"description"`
	FieldName   string `yaml:"fieldName"`
	Input       string `yaml:"input"`
	Want        string `yaml:"want"`
}

type checkBlockedCase struct {
	Description       string                                   `yaml:"description"`
	API               parserbase.ElasticsearchMaskableAPI      `yaml:"api"`
	Index             string                                   `yaml:"index"`
	BlockedFeatures   []parserbase.ElasticsearchBlockedFeature `yaml:"blockedFeatures"`
	WantError         bool                                     `yaml:"wantError"`
	WantErrorContains string                                   `yaml:"wantErrorContains"`
}

func loadElasticsearchTestData(t *testing.T) *elasticsearchTestData {
	t.Helper()
	f, err := os.Open("test-data/elasticsearch_masking.yaml")
	require.NoError(t, err)
	raw, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var data elasticsearchTestData
	require.NoError(t, yaml.Unmarshal(raw, &data))
	return &data
}

// requireJSONEqual compares two JSON strings by unmarshalling and comparing as Go values.
func requireJSONEqual(t *testing.T, want, got string) {
	t.Helper()
	var wantVal, gotVal any
	require.NoError(t, json.Unmarshal([]byte(want), &wantVal))
	require.NoError(t, json.Unmarshal([]byte(got), &gotVal))
	require.Equal(t, wantVal, gotVal)
}

func TestLookupSemanticTypeByDotPath(t *testing.T) {
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()

	for _, tc := range td.LookupSemanticType {
		t.Run(tc.Description, func(t *testing.T) {
			got := lookupSemanticTypeByDotPath(tc.DotPath, schema)
			require.Equal(t, tc.WantType, got)
		})
	}

	t.Run("nil schema", func(t *testing.T) {
		got := lookupSemanticTypeByDotPath("email", nil)
		require.Equal(t, "", got)
	})
}

func TestMaskElasticsearchHitsColumn(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskHitsColumn {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			result, err := maskElasticsearchHitsColumn(tc.Input, tc.SortFields, anyIndex(schema), maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchDocSource(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskDocSource {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			result, err := maskDocumentString(tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchGetSourceColumn(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskGetSourceColumn {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			result, err := maskElasticsearchGetSourceColumn(tc.FieldName, tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchMGetSource(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskMGetSource {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			result, err := maskElasticsearchMGetSource(tc.Input, anyIndex(schema), maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchInnerHits(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskInnerHits {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			result, err := maskElasticsearchHitsColumn(tc.Input, tc.SortFields, anyIndex(schema), maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchMSearchResponses(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskMSearchResponses {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			result, err := maskElasticsearchMSearchResponses(tc.Input, tc.SortFields, anyIndex(schema), maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchSourceObjectDirectReplacement(t *testing.T) {
	t.Parallel()
	maskers := map[string]masker.Masker{
		masker.DefaultSemanticTypeID: masker.NewDefaultFullMasker(),
	}

	tests := []struct {
		description string
		input       map[string]any
		schema      *storepb.ObjectSchema
		want        map[string]any
	}{
		{
			description: "tagged field with nested object is replaced directly",
			input: map[string]any{
				"name": "Alice",
				"contact": map[string]any{
					"phone": "555-1234",
					"city":  "NYC",
				},
			},
			schema: &storepb.ObjectSchema{
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"name": {Type: storepb.ObjectSchema_STRING},
							"contact": {
								Type:         storepb.ObjectSchema_OBJECT,
								SemanticType: masker.DefaultSemanticTypeID,
								Kind: &storepb.ObjectSchema_StructKind_{
									StructKind: &storepb.ObjectSchema_StructKind{
										Properties: map[string]*storepb.ObjectSchema{
											"phone": {Type: storepb.ObjectSchema_STRING},
											"city":  {Type: storepb.ObjectSchema_STRING},
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[string]any{
				"name":    "Alice",
				"contact": "******",
			},
		},
		{
			description: "tagged field with array value is replaced directly",
			input: map[string]any{
				"name": "Alice",
				"tags": []any{"tag1", "tag2"},
			},
			schema: &storepb.ObjectSchema{
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"name": {Type: storepb.ObjectSchema_STRING},
							"tags": {
								Type:         storepb.ObjectSchema_ARRAY,
								SemanticType: masker.DefaultSemanticTypeID,
							},
						},
					},
				},
			},
			want: map[string]any{
				"name": "Alice",
				"tags": "******",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			masked, err := walkAndMaskJSONRecursive(tc.input, tc.schema, maskers)
			require.NoError(t, err)
			got, ok := masked.(map[string]any)
			require.True(t, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCheckElasticsearchRequestBlocked(t *testing.T) {
	t.Parallel()
	td := loadElasticsearchTestData(t)

	for _, tc := range td.CheckBlocked {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			analysis := &parserbase.ElasticsearchAnalysis{
				API:             tc.API,
				Index:           tc.Index,
				BlockedFeatures: tc.BlockedFeatures,
			}
			err := checkElasticsearchRequestBlocked(analysis)
			if tc.WantError {
				require.Error(t, err)
				if tc.WantErrorContains != "" {
					require.Contains(t, err.Error(), tc.WantErrorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// testElasticsearchTables is a synced database of users (masked), orders (not
// masked) and the data stream logs-app (masked), with the alias people on
// users and the alias everything on users and orders.
func testElasticsearchTables() *elasticsearchTables {
	return newElasticsearchTables(
		&metadatapb.DatabaseSchemaMetadata{
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{Name: "logs-app"}, {Name: "orders"}, {Name: "users"}},
				Synonyms: []*metadatapb.SynonymMetadata{
					{Name: "everything", TargetName: "orders"},
					{Name: "everything", TargetName: "users"},
					{Name: "people", TargetName: "users"},
				},
			}},
		},
		&storepb.DatabaseConfig{
			Schemas: []*storepb.SchemaCatalog{{
				Tables: []*storepb.TableCatalog{
					{Name: "logs-app", ObjectSchema: testElasticsearchObjectSchema()},
					{Name: "orders", ObjectSchema: &storepb.ObjectSchema{Kind: &storepb.ObjectSchema_StructKind_{StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{"email": {Type: storepb.ObjectSchema_STRING}},
					}}}},
					{Name: "users", ObjectSchema: testElasticsearchObjectSchema()},
				},
			}},
		},
	)
}

func testElasticsearchSpan(t *testing.T, request string) *parserbase.QuerySpan {
	t.Helper()
	span, err := elasticsearchparser.GetQuerySpan(context.Background(), parserbase.GetQuerySpanContext{}, parserbase.Statement{Text: request}, "", "", false)
	require.NoError(t, err)
	require.NotNil(t, span.ElasticsearchAnalysis, request)
	return span
}

func TestElasticsearchResolveTargets(t *testing.T) {
	t.Parallel()
	tables := testElasticsearchTables()
	tests := []struct {
		request        string
		wantTables     []string
		wantUnresolved []string
	}{
		{request: "GET users/_search", wantTables: []string{"users"}},
		{request: "GET user*/_search", wantTables: []string{"users"}},
		{request: "GET people/_search", wantTables: []string{"users"}},
		{request: "GET everything/_search", wantTables: []string{"orders", "users"}},
		{request: "GET orders,people/_search", wantTables: []string{"orders", "users"}},
		{request: "GET orders%2Cusers/_search", wantTables: []string{"orders", "users"}},
		{request: "GET _all/_search", wantTables: []string{"logs-app", "orders", "users"}},
		{request: "GET */_search", wantTables: []string{"logs-app", "orders", "users"}},
		{request: "GET _search", wantTables: []string{"logs-app", "orders", "users"}},
		{request: "GET logs-app/_search", wantTables: []string{"logs-app"}},
		{request: "GET .ds-logs-app-2026.11.26-000001/_search", wantTables: []string{"logs-app"}},
		{request: "GET .ds-logs-app-000001/_doc/1", wantTables: []string{"logs-app"}},
		{request: "GET .fs-logs-app-2026.11.26-000002/_search", wantTables: []string{"logs-app"}},
		{request: "GET newindex/_search", wantUnresolved: []string{"newindex"}},
		{request: "GET remote:users/_search", wantUnresolved: []string{"remote:users"}},
		{request: "GET .ds-*/_search", wantUnresolved: []string{".ds-*"}},
		{request: "GET orders*/_search?expand_wildcards=all", wantTables: []string{"orders"}, wantUnresolved: []string{"orders*"}},
		{request: "GET _search?expand_wildcards=open,hidden", wantTables: []string{"logs-app", "orders", "users"}, wantUnresolved: []string{"_all"}},
		{
			request:    "GET orders/_mget\n{\"docs\":[{\"_id\":\"1\"},{\"_index\":\"people\",\"_id\":\"2\"}]}",
			wantTables: []string{"orders", "users"},
		},
		{
			request:    "GET _msearch\n{\"index\":\"orders\"}\n{}\n{\"index\":\"people\"}\n{}",
			wantTables: []string{"orders", "users"},
		},
		{
			request:        "GET orders/_msearch\n{\"index\":\"orders\",\"expand_wildcards\":\"all\"}\n{}\n{\"index\":\"o*\"}\n{}",
			wantTables:     []string{"orders"},
			wantUnresolved: []string{"o*"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.request, func(t *testing.T) {
			t.Parallel()
			gotTables, gotUnresolved := tables.resolve(testElasticsearchSpan(t, tc.request).ElasticsearchAnalysis)
			require.Equal(t, tc.wantTables, gotTables)
			require.Equal(t, tc.wantUnresolved, gotUnresolved)
		})
	}
}

func TestElasticsearchCheckRequests(t *testing.T) {
	t.Parallel()
	tables := testElasticsearchTables()
	tests := []struct {
		request      string
		wantContains string
	}{
		{request: "GET orders/_search\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}"},
		{request: "GET people/_search\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}", wantContains: "not supported when data masking"},
		{request: "GET user*/_search\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}", wantContains: "not supported when data masking"},
		{request: "GET orders,users/_search\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}", wantContains: "not supported when data masking"},
		{request: "GET _search\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}", wantContains: "not supported when data masking"},
		{request: "POST _sql\n{\"query\":\"SELECT email FROM users\"}", wantContains: "not supported when data masking"},
		{request: "POST _search/scroll\n{\"scroll_id\":\"x\"}", wantContains: "not supported when data masking"},
		{request: "GET .ds-logs-app-2026.11.26-000001/_search\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}", wantContains: "not supported when data masking"},
		{request: "GET orders/_msearch\n{\"index\":\"people\"}\n{\"aggs\":{\"a\":{\"terms\":{\"field\":\"email\"}}}}", wantContains: "not supported when data masking"},
		{request: "GET newindex/_search", wantContains: `cannot tell which indices "newindex" read`},
		{request: "GET .ds-*/_search", wantContains: `cannot tell which indices ".ds-*" read`},
		{request: "GET everything/_source/1", wantContains: "name one index"},
		{request: "GET people/_source/1"},
		{request: "PUT newindex/_doc/1\n{}"},
	}
	for _, tc := range tests {
		t.Run(tc.request, func(t *testing.T) {
			t.Parallel()
			err := tables.checkRequests([]*parserbase.QuerySpan{testElasticsearchSpan(t, tc.request)})
			if tc.wantContains == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantContains)
		})
	}

	t.Run("no masking", func(t *testing.T) {
		t.Parallel()
		unmasked := newElasticsearchTables(tables.metadata, nil)
		require.NoError(t, unmasked.checkRequests([]*parserbase.QuerySpan{testElasticsearchSpan(t, "GET newindex/_search")}))
	})
}

func TestElasticsearchMaskResultsByIndex(t *testing.T) {
	t.Parallel()
	tables := testElasticsearchTables()
	tests := []struct {
		description string
		request     string
		columns     []string
		values      []string
		want        []string
		wantError   string
	}{
		{
			description: "hits of a comma list are masked by their own index",
			request:     "GET orders,users/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":"orders","_source":{"email":"o@x"}},{"_index":"users","_source":{"email":"u@x","name":"u"}}]}`},
			want:        []string{`{"hits":[{"_index":"orders","_source":{"email":"o@x"}},{"_index":"users","_source":{"email":"******","name":"u"}}]}`},
		},
		{
			description: "hits of a wildcard",
			request:     "GET user*/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":"users","_source":{"email":"u@x"}}]}`},
			want:        []string{`{"hits":[{"_index":"users","_source":{"email":"******"}}]}`},
		},
		{
			description: "hits of an alias name the index",
			request:     "GET everything/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":"users","_source":{"email":"u@x"}},{"_index":"orders","_source":{"email":"o@x"}}]}`},
			want:        []string{`{"hits":[{"_index":"users","_source":{"email":"******"}},{"_index":"orders","_source":{"email":"o@x"}}]}`},
		},
		{
			description: "hits of a request with no index",
			request:     "GET _search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":"users","_source":{"email":"u@x"}}]}`},
			want:        []string{`{"hits":[{"_index":"users","_source":{"email":"******"}}]}`},
		},
		{
			description: "hits of _all",
			request:     "GET _all/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":"users","_source":{"email":"u@x"}}]}`},
			want:        []string{`{"hits":[{"_index":"users","_source":{"email":"******"}}]}`},
		},
		{
			description: "hits of a data stream name its backing index",
			request:     "GET logs-app/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":".ds-logs-app-2026.11.26-000001","_source":{"email":"l@x"}}]}`},
			want:        []string{`{"hits":[{"_index":".ds-logs-app-2026.11.26-000001","_source":{"email":"******"}}]}`},
		},
		{
			description: "a backing index read by name",
			request:     "GET .ds-logs-app-2026.11.26-000001/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":".ds-logs-app-2026.11.26-000001","_source":{"contact":{"phone":"1"}}}]}`},
			want:        []string{`{"hits":[{"_index":".ds-logs-app-2026.11.26-000001","_source":{"contact":{"phone":"******"}}}]}`},
		},
		{
			description: "a hit of an index the sync did not store refuses the result",
			request:     "GET orders*/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_index":"orders-new","_source":{"email":"n@x"}}]}`},
			wantError:   `index "orders-new", which is not synced`,
		},
		{
			description: "a hit that names no index refuses the result",
			request:     "GET users/_search",
			columns:     []string{"hits"},
			values:      []string{`{"hits":[{"_source":{"email":"u@x"}}]}`},
			wantError:   "which is not synced",
		},
		{
			description: "_msearch responses are masked by each hit's index",
			request:     "GET orders/_msearch\n{}\n{}\n{\"index\":\"people\"}\n{}",
			columns:     []string{"responses"},
			values:      []string{`[{"hits":{"hits":[{"_index":"orders","_source":{"email":"o@x"}}]}},{"hits":{"hits":[{"_index":"users","_source":{"email":"u@x"}}]}}]`},
			want:        []string{`[{"hits":{"hits":[{"_index":"orders","_source":{"email":"o@x"}}]}},{"hits":{"hits":[{"_index":"users","_source":{"email":"******"}}]}}]`},
		},
		{
			description: "_mget docs are masked by each doc's index",
			request:     "GET orders/_mget\n{\"docs\":[{\"_id\":\"1\"},{\"_index\":\"people\",\"_id\":\"2\"}]}",
			columns:     []string{"docs"},
			values:      []string{`[{"_index":"orders","_source":{"email":"o@x"}},{"_index":"users","_source":{"email":"u@x"}}]`},
			want:        []string{`[{"_index":"orders","_source":{"email":"o@x"}},{"_index":"users","_source":{"email":"******"}}]`},
		},
		{
			description: "a document read through an alias is masked by its _index",
			request:     "GET people/_doc/1",
			columns:     []string{"_id", "_index", "_source"},
			values:      []string{`"1"`, `"users"`, `{"email":"u@x"}`},
			want:        []string{`"1"`, `"users"`, `{"email":"******"}`},
		},
		{
			description: "a document of a backing index",
			request:     "GET .ds-logs-app-2026.11.26-000001/_doc/1",
			columns:     []string{"_index", "_source"},
			values:      []string{`".ds-logs-app-2026.11.26-000001"`, `{"email":"l@x"}`},
			want:        []string{`".ds-logs-app-2026.11.26-000001"`, `{"email":"******"}`},
		},
		{
			description: "a _source read through an alias",
			request:     "GET people/_source/1",
			columns:     []string{"email", "name"},
			values:      []string{`"u@x"`, `"u"`},
			want:        []string{`"******"`, `"u"`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			row := &v1pb.QueryRow{}
			for _, value := range tc.values {
				row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value}})
			}
			result := &v1pb.QueryResult{ColumnNames: tc.columns, Rows: []*v1pb.QueryRow{row}, RowsCount: 1}
			spans := []*parserbase.QuerySpan{testElasticsearchSpan(t, tc.request)}
			require.NoError(t, tables.maskResults(spans, []*v1pb.QueryResult{result}, testElasticsearchMaskerMap()))
			if tc.wantError != "" {
				require.Contains(t, result.Error, tc.wantError)
				require.Empty(t, result.Rows)
				return
			}
			require.Empty(t, result.Error)
			for i, want := range tc.want {
				requireJSONEqual(t, want, result.Rows[0].Values[i].GetStringValue())
			}
		})
	}
}
