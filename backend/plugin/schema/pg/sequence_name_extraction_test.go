package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractSequenceNameFromNextval(t *testing.T) {
	testCases := []struct {
		name         string
		defaultValue string
		expectedName string
		description  string
	}{
		{
			name:         "Simple sequence name with single quotes",
			defaultValue: "nextval('users_id_seq')",
			expectedName: "users_id_seq",
			description:  "Basic case with simple sequence name",
		},
		{
			name:         "Sequence name with schema qualification",
			defaultValue: "nextval('public.users_id_seq')",
			expectedName: "users_id_seq",
			description:  "Schema-qualified sequence name",
		},
		{
			name:         "Sequence with regclass cast",
			defaultValue: "nextval('users_id_seq'::regclass)",
			expectedName: "users_id_seq",
			description:  "Sequence with ::regclass type cast",
		},
		{
			name:         "Schema and regclass",
			defaultValue: "nextval('public.users_id_seq'::regclass)",
			expectedName: "users_id_seq",
			description:  "Schema-qualified sequence with regclass cast",
		},
		{
			name:         "Quoted sequence name",
			defaultValue: `nextval('"Users_Id_Seq"')`,
			expectedName: "Users_Id_Seq",
			description:  "Sequence name with double quotes for case sensitivity",
		},
		{
			name:         "Quoted schema and sequence",
			defaultValue: `nextval('"Public"."Users_Id_Seq"')`,
			expectedName: "Users_Id_Seq",
			description:  "Both schema and sequence quoted",
		},
		{
			name:         "Mixed case with public schema",
			defaultValue: `nextval('public."MixedCaseSeq"')`,
			expectedName: "MixedCaseSeq",
			description:  "Unquoted schema with quoted sequence",
		},
		{
			name:         "Quoted schema unquoted sequence",
			defaultValue: `nextval('"Public".users_id_seq')`,
			expectedName: "users_id_seq",
			description:  "Quoted schema with unquoted sequence",
		},
		{
			name:         "Complex case with all features",
			defaultValue: `nextval('"MySchema"."MySeq"'::regclass)`,
			expectedName: "MySeq",
			description:  "Quoted schema, quoted sequence, and regclass",
		},
		{
			name:         "Uppercase NEXTVAL",
			defaultValue: `NEXTVAL('users_id_seq')`,
			expectedName: "users_id_seq",
			description:  "Case-insensitive NEXTVAL function name",
		},
		{
			name:         "Mixed case nextval",
			defaultValue: `NextVal('users_id_seq')`,
			expectedName: "users_id_seq",
			description:  "Mixed case nextval function name",
		},
		{
			name:         "Sequence with spaces",
			defaultValue: `nextval( 'users_id_seq' )`,
			expectedName: "users_id_seq",
			description:  "Extra spaces around sequence name",
		},
		{
			name:         "Double quoted outer quotes",
			defaultValue: `nextval("users_id_seq")`,
			expectedName: "users_id_seq",
			description:  "Double quotes as outer quotes",
		},
		{
			name:         "Empty default",
			defaultValue: "",
			expectedName: "",
			description:  "Empty string should return empty",
		},
		{
			name:         "No nextval",
			defaultValue: "42",
			expectedName: "",
			description:  "Non-nextval default should return empty",
		},
		{
			name:         "Malformed nextval",
			defaultValue: "nextval(",
			expectedName: "",
			description:  "Malformed nextval should return empty",
		},
		{
			name:         "Special characters in schema",
			defaultValue: `nextval('"schema-name"."seq_name"')`,
			expectedName: "seq_name",
			description:  "Schema with special characters",
		},
		{
			name:         "Multiple dots",
			defaultValue: `nextval('"my.schema"."my.seq"')`,
			expectedName: "my.seq",
			description:  "Dots within quoted identifiers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractSequenceNameFromNextval(tc.defaultValue)
			assert.Equal(t, tc.expectedName, result, tc.description)
		})
	}
}

func TestExtractIdentifierFromQualifiedName(t *testing.T) {
	testCases := []struct {
		name          string
		qualifiedName string
		expected      string
		description   string
	}{
		{
			name:          "Simple identifier",
			qualifiedName: "users_id_seq",
			expected:      "users_id_seq",
			description:   "No schema qualification",
		},
		{
			name:          "Schema qualified",
			qualifiedName: "public.users_id_seq",
			expected:      "users_id_seq",
			description:   "Simple schema.sequence",
		},
		{
			name:          "Both quoted",
			qualifiedName: `"Public"."Users_Id_Seq"`,
			expected:      `"Users_Id_Seq"`,
			description:   "Both parts quoted",
		},
		{
			name:          "Only sequence quoted",
			qualifiedName: `public."MixedCase"`,
			expected:      `"MixedCase"`,
			description:   "Only sequence name quoted",
		},
		{
			name:          "Dot in quoted identifier",
			qualifiedName: `"my.schema"."my.seq"`,
			expected:      `"my.seq"`,
			description:   "Dots inside quoted parts should be ignored",
		},
		{
			name:          "Empty string",
			qualifiedName: "",
			expected:      "",
			description:   "Empty input should return empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractIdentifierFromQualifiedName(tc.qualifiedName)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}
