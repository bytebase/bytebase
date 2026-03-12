package v1

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mongoparser "github.com/bytebase/bytebase/backend/plugin/parser/mongodb"
)

type mongodbMaskingTestData struct {
	MaskDocument []mongodbMaskDocumentCase `yaml:"maskDocument"`
	CheckBlocked []mongodbCheckBlockedCase `yaml:"checkBlocked"`
}

type mongodbMaskDocumentCase struct {
	Description string `yaml:"description"`
	Schema      string `yaml:"schema"`
	Input       string `yaml:"input"`
	Want        string `yaml:"want"`
}

type mongodbCheckBlockedCase struct {
	Description       string `yaml:"description"`
	AnalysisNil       bool   `yaml:"analysisNil"`
	API               string `yaml:"api"`
	Operation         string `yaml:"operation"`
	Collection        string `yaml:"collection"`
	UnsupportedStage  string `yaml:"unsupportedStage"`
	WantError         bool   `yaml:"wantError"`
	WantErrorContains string `yaml:"wantErrorContains"`
}

func loadMongoDBMaskingTestData(t *testing.T) *mongodbMaskingTestData {
	t.Helper()
	f, err := os.Open("test-data/mongodb_masking.yaml")
	require.NoError(t, err)
	raw, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var data mongodbMaskingTestData
	require.NoError(t, yaml.Unmarshal(raw, &data))
	return &data
}

func mustMongoSchema(t *testing.T, schemaName string) *storepb.ObjectSchema {
	t.Helper()
	switch schemaName {
	case "nested":
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"name": {Type: storepb.ObjectSchema_STRING},
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
	case "array":
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"tags": {
							Type: storepb.ObjectSchema_ARRAY,
							Kind: &storepb.ObjectSchema_ArrayKind_{
								ArrayKind: &storepb.ObjectSchema_ArrayKind{
									Kind: &storepb.ObjectSchema{Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
								},
							},
						},
						"count": {Type: storepb.ObjectSchema_NUMBER},
					},
				},
			},
		}
	case "tagged-node":
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"profile": {
							Type:         storepb.ObjectSchema_OBJECT,
							SemanticType: "bb.default",
							Kind: &storepb.ObjectSchema_StructKind_{
								StructKind: &storepb.ObjectSchema_StructKind{
									Properties: map[string]*storepb.ObjectSchema{
										"ssn":     {Type: storepb.ObjectSchema_STRING},
										"address": {Type: storepb.ObjectSchema_STRING},
									},
								},
							},
						},
						"name": {Type: storepb.ObjectSchema_STRING},
					},
				},
			},
		}
	case "extended-json":
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"_id": {
							Type: storepb.ObjectSchema_OBJECT,
							Kind: &storepb.ObjectSchema_StructKind_{
								StructKind: &storepb.ObjectSchema_StructKind{
									Properties: map[string]*storepb.ObjectSchema{
										"$oid": {Type: storepb.ObjectSchema_STRING},
									},
								},
							},
						},
						"email": {Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
						"createdAt": {
							Type: storepb.ObjectSchema_OBJECT,
							Kind: &storepb.ObjectSchema_StructKind_{
								StructKind: &storepb.ObjectSchema_StructKind{
									Properties: map[string]*storepb.ObjectSchema{
										"$date": {Type: storepb.ObjectSchema_STRING},
									},
								},
							},
						},
					},
				},
			},
		}
	case "array-level-masked":
		// tags array is masked at the array level (the whole array, or unwound scalar, is masked).
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"tags": {
							Type:         storepb.ObjectSchema_ARRAY,
							SemanticType: "bb.default",
							Kind: &storepb.ObjectSchema_ArrayKind_{
								ArrayKind: &storepb.ObjectSchema_ArrayKind{
									Kind: &storepb.ObjectSchema{Type: storepb.ObjectSchema_STRING},
								},
							},
						},
						"count": {Type: storepb.ObjectSchema_NUMBER},
					},
				},
			},
		}
	case "array-of-objects":
		// friends is an array of objects where phone is sensitive.
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"friends": {
							Type: storepb.ObjectSchema_ARRAY,
							Kind: &storepb.ObjectSchema_ArrayKind_{
								ArrayKind: &storepb.ObjectSchema_ArrayKind{
									Kind: &storepb.ObjectSchema{
										Type: storepb.ObjectSchema_OBJECT,
										Kind: &storepb.ObjectSchema_StructKind_{
											StructKind: &storepb.ObjectSchema_StructKind{
												Properties: map[string]*storepb.ObjectSchema{
													"name":  {Type: storepb.ObjectSchema_STRING},
													"phone": {Type: storepb.ObjectSchema_STRING, SemanticType: "bb.default"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	case "lookup-joined":
		// users collection with an injected "orders" field (array of objects where total is sensitive).
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{
					Properties: map[string]*storepb.ObjectSchema{
						"name": {Type: storepb.ObjectSchema_STRING},
						"orders": {
							Type: storepb.ObjectSchema_ARRAY,
							Kind: &storepb.ObjectSchema_ArrayKind_{
								ArrayKind: &storepb.ObjectSchema_ArrayKind{
									Kind: &storepb.ObjectSchema{
										Type: storepb.ObjectSchema_OBJECT,
										Kind: &storepb.ObjectSchema_StructKind_{
											StructKind: &storepb.ObjectSchema_StructKind{
												Properties: map[string]*storepb.ObjectSchema{
													"total":  {Type: storepb.ObjectSchema_NUMBER, SemanticType: "bb.default"},
													"status": {Type: storepb.ObjectSchema_STRING},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	case "non-object":
		return &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_OBJECT,
			Kind: &storepb.ObjectSchema_StructKind_{
				StructKind: &storepb.ObjectSchema_StructKind{},
			},
		}
	default:
		require.Failf(t, "unknown schema", "schema %q not found", schemaName)
		return nil
	}
}

func mustMaskableAPI(t *testing.T, api string) mongoparser.MaskableAPI {
	t.Helper()
	switch api {
	case "unsupported":
		return mongoparser.MaskableAPIUnsupported
	case "find":
		return mongoparser.MaskableAPIFind
	case "findOne":
		return mongoparser.MaskableAPIFindOne
	case "unsupportedRead":
		return mongoparser.MaskableAPIUnsupportedRead
	case "aggregate":
		return mongoparser.MaskableAPIAggregate
	default:
		require.Failf(t, "unknown api", "api %q not found", api)
		return mongoparser.MaskableAPIUnsupported
	}
}

func TestMaskMongoDBDocumentString(t *testing.T) {
	td := loadMongoDBMaskingTestData(t)
	maskers := map[string]masker.Masker{
		"bb.default": masker.NewDefaultFullMasker(),
	}

	for _, tc := range td.MaskDocument {
		t.Run(tc.Description, func(t *testing.T) {
			schema := mustMongoSchema(t, tc.Schema)
			got, err := maskDocumentString(tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, got)
		})
	}
}

func TestCheckMongoDBRequestBlocked(t *testing.T) {
	td := loadMongoDBMaskingTestData(t)

	for _, tc := range td.CheckBlocked {
		t.Run(tc.Description, func(t *testing.T) {
			var analysis *mongoparser.MaskingAnalysis
			if !tc.AnalysisNil {
				analysis = &mongoparser.MaskingAnalysis{
					API:              mustMaskableAPI(t, tc.API),
					Operation:        tc.Operation,
					Collection:       tc.Collection,
					UnsupportedStage: tc.UnsupportedStage,
				}
			}
			err := checkMongoDBRequestBlocked(analysis)
			if tc.WantError {
				require.Error(t, err)
				if tc.WantErrorContains != "" {
					require.Contains(t, err.Error(), tc.WantErrorContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
