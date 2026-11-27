package v1

import (
	"context"
	"io"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	mongodbparser "github.com/bytebase/bytebase/backend/plugin/parser/mongodb"
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
									Kind: &storepb.ObjectSchema{Type: storepb.ObjectSchema_STRING, SemanticType: masker.DefaultSemanticTypeID},
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
							SemanticType: masker.DefaultSemanticTypeID,
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
						"email": {Type: storepb.ObjectSchema_STRING, SemanticType: masker.DefaultSemanticTypeID},
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
							SemanticType: masker.DefaultSemanticTypeID,
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
													"phone": {Type: storepb.ObjectSchema_STRING, SemanticType: masker.DefaultSemanticTypeID},
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
													"total":  {Type: storepb.ObjectSchema_NUMBER, SemanticType: masker.DefaultSemanticTypeID},
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

func mustMaskableAPI(t *testing.T, api string) parserbase.MongoDBMaskableAPI {
	t.Helper()
	switch api {
	case "unsupported":
		return parserbase.MongoDBMaskableAPIUnsupported
	case "find":
		return parserbase.MongoDBMaskableAPIFind
	case "findOne":
		return parserbase.MongoDBMaskableAPIFindOne
	case "unsupportedRead":
		return parserbase.MongoDBMaskableAPIUnsupportedRead
	case "aggregate":
		return parserbase.MongoDBMaskableAPIAggregate
	default:
		require.Failf(t, "unknown api", "api %q not found", api)
		return parserbase.MongoDBMaskableAPIUnsupported
	}
}

func TestMaskMongoDBDocumentString(t *testing.T) {
	t.Parallel()
	td := loadMongoDBMaskingTestData(t)
	maskers := map[string]masker.Masker{
		masker.DefaultSemanticTypeID: masker.NewDefaultFullMasker(),
	}

	for _, tc := range td.MaskDocument {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			schema := mustMongoSchema(t, tc.Schema)
			got, err := maskDocumentString(tc.Input, schema, maskers)
			require.NoError(t, err)
			requireJSONEqual(t, tc.Want, got)
		})
	}
}

func TestCheckMongoDBRequestBlocked(t *testing.T) {
	t.Parallel()
	td := loadMongoDBMaskingTestData(t)

	for _, tc := range td.CheckBlocked {
		t.Run(tc.Description, func(t *testing.T) {
			t.Parallel()
			var analysis *parserbase.MongoDBAnalysis
			if !tc.AnalysisNil {
				analysis = &parserbase.MongoDBAnalysis{
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

// testMongoDBDatabases is an instance where mydb holds users (masked), orders
// (no schema) and the view usersView on users, and other holds users (masked).
func testMongoDBDatabases() *mongoDBDatabases {
	masked := &storepb.DatabaseConfig{Schemas: []*storepb.SchemaCatalog{{
		Tables: []*storepb.TableCatalog{{Name: "users", ObjectSchema: testMongoDBUsersSchema()}},
	}}}
	databases := map[string]*metadatapb.DatabaseSchemaMetadata{
		"mydb": {Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{Name: "orders"}, {Name: "users"}},
			Views:  []*metadatapb.ViewMetadata{{Name: "usersView", Definition: `{"viewOn": "users", "pipeline": []}`}},
		}}},
		"other": {Schemas: []*metadatapb.SchemaMetadata{{Tables: []*metadatapb.TableMetadata{{Name: "users"}}}}},
	}
	return &mongoDBDatabases{
		current: "mydb",
		load: func(name string) (*metadatapb.DatabaseSchemaMetadata, *storepb.DatabaseConfig, error) {
			if databases[name] == nil {
				return nil, nil, nil
			}
			return databases[name], masked, nil
		},
		loaded: map[string]*mongoDBDatabase{},
	}
}

func testMongoDBUsersSchema() *storepb.ObjectSchema {
	return &storepb.ObjectSchema{
		Type: storepb.ObjectSchema_OBJECT,
		Kind: &storepb.ObjectSchema_StructKind_{StructKind: &storepb.ObjectSchema_StructKind{
			Properties: map[string]*storepb.ObjectSchema{
				"name":  {Type: storepb.ObjectSchema_STRING},
				"email": {Type: storepb.ObjectSchema_STRING, SemanticType: masker.DefaultSemanticTypeID},
			},
		}},
	}
}

func TestMongoDBCheckReads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		statement string
		wantError string
	}{
		{statement: `db.users.find()`},
		{statement: `db.orders.find()`},
		{statement: `db.orders.aggregate([{$lookup: {from: "users", localField: "u", foreignField: "_id", as: "u"}}])`},
		{statement: `db.orders.aggregate([{$graphLookup: {from: "users", startWith: "$u", connectFromField: "u", connectToField: "_id", as: "u"}}])`},
		{statement: `db.orders.aggregate([{$unionWith: "orders"}])`},
		{statement: `db.orders.aggregate([{$unionWith: "users"}])`, wantError: `collection "users" of database "mydb"`},
		{statement: `db.orders.aggregate([{$unionWith: {coll: "users", pipeline: [{$match: {}}]}}])`, wantError: `collection "users"`},
		{statement: `db.orders.aggregate([{$facet: {a: [{$lookup: {from: "users", localField: "u", foreignField: "_id", as: "u"}}]}}])`, wantError: `collection "users"`},
		{statement: `db.orders.aggregate([{$lookup: {from: "users", pipeline: [{$match: {}}], as: "u"}}])`, wantError: `collection "users"`},
		{statement: `db.orders.aggregate([{$lookup: {from: "users", localField: "u", foreignField: "_id", as: "a.b"}}])`, wantError: `collection "users"`},
		{statement: `db.orders.aggregate([{$lookup: {from: {db: "mydb", coll: "users"}, localField: "u", foreignField: "_id", as: "u"}}])`, wantError: `collection "users" of database "mydb"`},
		{statement: `db.orders.aggregate([{$lookup: {from: {db: "other", coll: "users"}, localField: "u", foreignField: "_id", as: "u"}}])`, wantError: `collection "users" of database "other"`},
		{
			statement: `db.orders.aggregate([{$lookup: {from: "users", localField: "u", foreignField: "_id", as: "u"}}, {$lookup: {from: {db: "mydb", coll: "users"}, localField: "v", foreignField: "_id", as: "v"}}])`,
			wantError: `collection "users"`,
		},
		{statement: `db.usersView.find()`, wantError: `collection "users"`},
		{statement: `db.orders.aggregate([{$lookup: {from: "usersView", localField: "u", foreignField: "_id", as: "u"}}])`, wantError: `collection "users"`},
		{statement: `db.users.aggregate([{$lookup: {from: "usersView", localField: "u", foreignField: "_id", as: "u"}}])`, wantError: `collection "users"`},
		{statement: `db.runCommand({aggregate: "orders", pipeline: [{$unionWith: "users"}], cursor: {}})`, wantError: `collection "users"`},
		{statement: `db.users.distinct("email")`, wantError: `collection "users"`},
	}
	for _, tc := range tests {
		t.Run(tc.statement, func(t *testing.T) {
			t.Parallel()
			span, err := mongodbparser.GetQuerySpan(context.Background(), parserbase.GetQuerySpanContext{}, parserbase.Statement{Text: tc.statement}, "mydb", "", false)
			require.NoError(t, err)
			require.NotNil(t, span.MongoDBAnalysis)
			err = testMongoDBDatabases().checkReads(span.MongoDBAnalysis)
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			var refusal *errMongoDBUncoveredRead
			require.ErrorAs(t, err, &refusal)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestMongoDBJoinMaskedWithoutSourceSchema(t *testing.T) {
	t.Parallel()
	maskers := map[string]masker.Masker{masker.DefaultSemanticTypeID: masker.NewDefaultFullMasker()}
	for _, source := range []*storepb.ObjectSchema{nil, {Type: storepb.ObjectSchema_OBJECT}} {
		schema := injectJoinedSchemas(source, []joinedSchema{{asField: "u", schema: testMongoDBUsersSchema()}})
		got, err := maskDocumentString(`{"item": "x", "u": [{"name": "n", "email": "e@x"}]}`, schema, maskers)
		require.NoError(t, err)
		requireJSONEqual(t, `{"item": "x", "u": [{"name": "n", "email": "******"}]}`, got)
	}
}

func TestMongoDBPredicateRefusal(t *testing.T) {
	t.Parallel()
	orders := &storepb.ObjectSchema{Type: storepb.ObjectSchema_OBJECT}
	joinedOverEmail := injectJoinedSchemas(testMongoDBUsersSchema(), []joinedSchema{{asField: "email", schema: orders}})
	joinedUsers := injectJoinedSchemas(nil, []joinedSchema{{asField: "u", schema: testMongoDBUsersSchema()}})
	tests := []struct {
		statement string
		source    *storepb.ObjectSchema
		result    *storepb.ObjectSchema
		want      string
	}{
		{
			// The join overwrites email, but the $match before it reads the source field.
			statement: `db.users.aggregate([{$match: {email: "guess@example.com"}}, {$lookup: {from: "orders", localField: "_id", foreignField: "userId", as: "email"}}])`,
			source:    testMongoDBUsersSchema(),
			result:    joinedOverEmail,
			want:      `using field "email"`,
		},
		{
			statement: `db.orders.aggregate([{$lookup: {from: "users", localField: "u", foreignField: "_id", as: "u"}}, {$match: {"u.email": "guess@example.com"}}])`,
			result:    joinedUsers,
			want:      `using field "u.email"`,
		},
		{
			statement: `db.orders.aggregate([{$lookup: {from: "users", localField: "u", foreignField: "_id", as: "u"}}, {$match: {u: {$elemMatch: {name: "n"}}}}])`,
			result:    injectJoinedSchemas(nil, []joinedSchema{{asField: "u", schema: &storepb.ObjectSchema{Type: storepb.ObjectSchema_OBJECT, SemanticType: masker.DefaultSemanticTypeID}}}),
			want:      `using field "u"`,
		},
		{
			statement: `db.users.find({name: "n"})`,
			source:    testMongoDBUsersSchema(),
			result:    testMongoDBUsersSchema(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.statement, func(t *testing.T) {
			t.Parallel()
			span, err := mongodbparser.GetQuerySpan(context.Background(), parserbase.GetQuerySpanContext{}, parserbase.Statement{Text: tc.statement}, "mydb", "", false)
			require.NoError(t, err)
			got := mongoDBPredicateRefusal(span, tc.source, tc.result)
			if tc.want == "" {
				require.Empty(t, got)
				return
			}
			require.Contains(t, got, tc.want)
		})
	}
}
