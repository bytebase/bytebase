package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetFirstSemanticTypeInPath(t *testing.T) {
	containerName := "container"
	containerNode := base.NewItemSelector(containerName)

	testCases := []struct {
		nodes        []base.SelectorNode
		objectSchema *storepb.ObjectSchema
		want         string
	}{
		{
			nodes: []base.SelectorNode{
				base.NewItemSelector("a"),
			},
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"a": {
								SemanticType: "st-a",
								Type:         storepb.ObjectSchema_STRING,
							},
						},
					},
				},
			},
			want: "st-a",
		},
		{
			nodes: []base.SelectorNode{
				base.NewItemSelector("a"),
				base.NewArraySelector("b", 1),
				base.NewItemSelector("c"),
			},
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"a": {
								Type: storepb.ObjectSchema_OBJECT,
								Kind: &storepb.ObjectSchema_StructKind_{
									StructKind: &storepb.ObjectSchema_StructKind{
										Properties: map[string]*storepb.ObjectSchema{
											"b": {
												Type:         storepb.ObjectSchema_ARRAY,
												SemanticType: "st-b",
												Kind: &storepb.ObjectSchema_ArrayKind_{
													ArrayKind: &storepb.ObjectSchema_ArrayKind{
														Kind: &storepb.ObjectSchema{
															Type: storepb.ObjectSchema_OBJECT,
															Kind: &storepb.ObjectSchema_StructKind_{
																StructKind: &storepb.ObjectSchema_StructKind{
																	Properties: map[string]*storepb.ObjectSchema{
																		"c": {
																			Type:         storepb.ObjectSchema_STRING,
																			SemanticType: "st-c",
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
								},
							},
						},
					},
				},
			},
			want: "st-c",
		},
		{
			nodes: []base.SelectorNode{
				base.NewItemSelector("a"),
				base.NewItemSelector("c"),
			},
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"a": {
								Type: storepb.ObjectSchema_OBJECT,
								Kind: &storepb.ObjectSchema_StructKind_{
									StructKind: &storepb.ObjectSchema_StructKind{
										Properties: map[string]*storepb.ObjectSchema{
											"b": {
												Type:         storepb.ObjectSchema_STRING,
												SemanticType: "st-b",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: "",
		},
	}

	for _, tc := range testCases {
		if len(tc.nodes) == 0 {
			continue
		}

		ast := base.NewPathAST(containerNode)
		next := ast.Root
		for i := 0; i < len(tc.nodes); i++ {
			next.SetNext(tc.nodes[i])
			next = next.GetNext()
		}

		got := getFirstSemanticTypeInPath(ast, tc.objectSchema)
		require.Equal(t, tc.want, got)
	}
}

func TestWalkAndMaskJSON(t *testing.T) {
	type testCase struct {
		description          string
		input                string
		objectSchema         *storepb.ObjectSchema
		semanticTypeToMasker map[string]masker.Masker
		want                 string
	}

	testCases := []testCase{
		{
			description: "empty object",
			input:       `{}`,
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"ssn": {
								SemanticType: "PII-SSN",
								Type:         storepb.ObjectSchema_STRING,
							},
						},
					},
				},
			},
			semanticTypeToMasker: map[string]masker.Masker{},
			want:                 `{}`,
		},
		{
			description:          "no semantic type",
			input:                `{"name": "John"}`,
			objectSchema:         &storepb.ObjectSchema{},
			semanticTypeToMasker: map[string]masker.Masker{},
			want:                 `{"name": "John"}`,
		},
		{
			description: "mask the outest semantic type",
			input:       `{"name": "John", "ssn": "123-45-6789"}`,
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"ssn": {
								SemanticType: "PII-SSN",
								Type:         storepb.ObjectSchema_STRING,
							},
						},
					},
				},
			},
			semanticTypeToMasker: map[string]masker.Masker{
				"PII-SSN": masker.NewFullMasker("******"),
			},
			want: `{"name": "John", "ssn": "******"}`,
		},
		{
			description: "mask the inner semantic type",
			input:       `{"name": "John", "information": {"ssn": "123-45-6789"}}`,
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"information": {
								Type: storepb.ObjectSchema_OBJECT,
								Kind: &storepb.ObjectSchema_StructKind_{
									StructKind: &storepb.ObjectSchema_StructKind{
										Properties: map[string]*storepb.ObjectSchema{
											"ssn": {
												Type:         storepb.ObjectSchema_STRING,
												SemanticType: "PII-SSN",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			semanticTypeToMasker: map[string]masker.Masker{
				"PII-SSN": masker.NewFullMasker("******"),
			},
			want: `{"name": "John", "information": {"ssn": "******"}}`,
		},
		{
			description: "Recursive apply the masker to the object",
			input:       `{"name": "John", "information": {"ssn": "123-45-6789"}}`,
			objectSchema: &storepb.ObjectSchema{
				Type:         storepb.ObjectSchema_OBJECT,
				SemanticType: "SSN",
			},
			semanticTypeToMasker: map[string]masker.Masker{
				"SSN": masker.NewFullMasker("******"),
			},
			want: `{"name": "******", "information": {"ssn": "******"}}`,
		},
		{
			description: "Mask the array",
			input:       `{"name": "John", "information": ["123-45-6789", "987-65-4321"]}`,
			objectSchema: &storepb.ObjectSchema{
				Type: storepb.ObjectSchema_OBJECT,
				Kind: &storepb.ObjectSchema_StructKind_{
					StructKind: &storepb.ObjectSchema_StructKind{
						Properties: map[string]*storepb.ObjectSchema{
							"information": {
								Type: storepb.ObjectSchema_ARRAY,
								Kind: &storepb.ObjectSchema_ArrayKind_{
									ArrayKind: &storepb.ObjectSchema_ArrayKind{
										Kind: &storepb.ObjectSchema{
											Type:         storepb.ObjectSchema_STRING,
											SemanticType: "SSN",
										},
									},
								},
							},
						},
					},
				},
			},
			semanticTypeToMasker: map[string]masker.Masker{
				"SSN": masker.NewFullMasker("******"),
			},
			want: `{"name": "John", "information": ["******", "******"]}`,
		},
	}

	for _, tc := range testCases {
		var input any
		err := json.Unmarshal([]byte(tc.input), &input)
		require.NoError(t, err, tc.description)

		got, err := walkAndMaskJSON(input, tc.objectSchema, tc.semanticTypeToMasker)
		require.NoError(t, err, tc.description)

		output, err := json.Marshal(got)
		require.NoError(t, err, tc.description)

		require.NoError(t, err, tc.description)
		require.JSONEq(t, tc.want, string(output))
	}
}
