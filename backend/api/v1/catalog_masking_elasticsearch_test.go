package v1

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	esparser "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
)

// testElasticsearchObjectSchema creates a schema where "email" and "contact.phone" are masked.
func testElasticsearchObjectSchema() *storepb.ObjectSchema {
	return &storepb.ObjectSchema{
		Kind: &storepb.ObjectSchema_StructKind_{
			StructKind: &storepb.ObjectSchema_StructKind{
				Properties: map[string]*storepb.ObjectSchema{
					"name":  {Type: storepb.ObjectSchema_STRING},
					"email": {Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
					"age":   {Type: storepb.ObjectSchema_NUMBER},
					"contact": {
						Type: storepb.ObjectSchema_OBJECT,
						Kind: &storepb.ObjectSchema_StructKind_{
							StructKind: &storepb.ObjectSchema_StructKind{
								Properties: map[string]*storepb.ObjectSchema{
									"phone": {Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
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

func testElasticsearchMaskerMap() map[string]masker.Masker {
	return map[string]masker.Masker{
		"bb.default": masker.NewDefaultFullMasker(),
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
	Description       string                    `yaml:"description"`
	API               esparser.MaskableAPI      `yaml:"api"`
	Index             string                    `yaml:"index"`
	BlockedFeatures   []esparser.BlockedFeature `yaml:"blockedFeatures"`
	WantError         bool                      `yaml:"wantError"`
	WantErrorContains string                    `yaml:"wantErrorContains"`
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
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskHitsColumn {
		t.Run(tc.Description, func(t *testing.T) {
			result, err := maskElasticsearchHitsColumn(tc.Input, tc.SortFields, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchDocSource(t *testing.T) {
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskDocSource {
		t.Run(tc.Description, func(t *testing.T) {
			result, err := maskElasticsearchDocSource(tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchGetSourceColumn(t *testing.T) {
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskGetSourceColumn {
		t.Run(tc.Description, func(t *testing.T) {
			result, err := maskElasticsearchGetSourceColumn(tc.FieldName, tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchMGetSource(t *testing.T) {
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskMGetSource {
		t.Run(tc.Description, func(t *testing.T) {
			result, err := maskElasticsearchMGetSource(tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchInnerHits(t *testing.T) {
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskInnerHits {
		t.Run(tc.Description, func(t *testing.T) {
			result, err := maskElasticsearchHitsColumn(tc.Input, tc.SortFields, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchMSearchResponses(t *testing.T) {
	td := loadElasticsearchTestData(t)
	schema := testElasticsearchObjectSchema()
	maskers := testElasticsearchMaskerMap()

	for _, tc := range td.MaskMSearchResponses {
		t.Run(tc.Description, func(t *testing.T) {
			result, err := maskElasticsearchMSearchResponses(tc.Input, tc.SortFields, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, result)
		})
	}
}

func TestMaskElasticsearchSourceObjectDirectReplacement(t *testing.T) {
	maskers := map[string]masker.Masker{
		"bb.default": masker.NewDefaultFullMasker(),
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
								SemanticType: "bb.default",
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
								SemanticType: "bb.default",
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
			got, err := maskElasticsearchSourceObject(tc.input, tc.schema, maskers)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCheckElasticsearchRequestBlocked(t *testing.T) {
	td := loadElasticsearchTestData(t)

	for _, tc := range td.CheckBlocked {
		t.Run(tc.Description, func(t *testing.T) {
			analysis := &esparser.RequestAnalysis{
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
