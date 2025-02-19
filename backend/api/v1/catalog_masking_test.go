package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
				SemanticType: "FULL",
			},
			semanticTypeToMasker: map[string]masker.Masker{
				"FULL": masker.NewFullMasker("******"),
			},
			want: `{"name": "******", "information": {"ssn": "******"}}`,
		},
	}

	for _, tc := range testCases {
		var input any
		err := json.Unmarshal([]byte(tc.input), &input)
		require.NoError(t, err)

		got, err := walkAndMaskJSON(input, tc.objectSchema, tc.semanticTypeToMasker)
		require.NoError(t, err)

		output, err := json.Marshal(got)
		require.NoError(t, err)

		require.NoError(t, err)
		require.JSONEq(t, tc.want, string(output))
	}
}
