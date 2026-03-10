package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestParsePGConfiguredSearchPath(t *testing.T) {
	got := ParsePGConfiguredSearchPath(` "$user" , public, "CamelCase", "schema,with,comma", pg_catalog, 'SingleQuoted', information_schema `)

	require.Equal(t, []PGSearchPathItem{
		{CurrentUser: true},
		{Schema: "public"},
		{Schema: "CamelCase"},
		{Schema: "schema,with,comma"},
		{Schema: "SingleQuoted"},
	}, got)
}

func TestResolvePGSearchPath(t *testing.T) {
	configured := []PGSearchPathItem{
		{CurrentUser: true},
		{Schema: "public"},
		{Schema: "MissingSchema"},
		{Schema: "CamelCase"},
	}

	schemaSet := map[string]bool{
		"alice":     true,
		"public":    true,
		"CamelCase": true,
	}
	got := ResolvePGSearchPath(configured, "alice", func(schema string) bool {
		return schemaSet[schema]
	})

	require.Equal(t, []string{"alice", "public", "CamelCase"}, got)
}

func TestDatabaseMetadataSearchPathHelpers(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name:       "testdb",
		SearchPath: `"$user", public, MissingSchema, "CamelCase"`,
		Schemas: []*storepb.SchemaMetadata{
			{Name: "alice"},
			{Name: "public"},
			{Name: "CamelCase"},
		},
	}

	db := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)

	require.Equal(t, []PGSearchPathItem{
		{CurrentUser: true},
		{Schema: "public"},
		{Schema: "missingschema"},
		{Schema: "CamelCase"},
	}, db.GetConfiguredSearchPath())
	require.Equal(t, []string{"public", "missingschema", "CamelCase"}, db.GetSearchPath())
	require.Equal(t, []string{"alice", "public", "CamelCase"}, db.GetSearchPathForCurrentUser("alice"))
}

func TestDatabaseMetadataSearchPathHelpersSkipMissingCurrentUserSchema(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name:       "testdb",
		SearchPath: `"$user", public, MissingSchema`,
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}

	db := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)

	require.Equal(t, []string{"public"}, db.GetSearchPathForCurrentUser("alice"))
}
