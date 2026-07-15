package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinitionSDLFormat_CompositeTypes(t *testing.T) {
	// aa_nested sorts before its dependency zz_base alphabetically, so correct
	// output proves dependency ordering rather than name ordering.
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "order_status", Values: []string{"pending", "shipped"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "aa_nested",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "home", Type: "public.zz_base"},
							{Name: "homes", Type: "public.zz_base[]"},
							{Name: "status", Type: "public.order_status"},
						},
					},
					{
						Name:    "zz_base",
						Comment: "base address type",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text", Collation: "C", Comment: "street line"},
							{Name: "city", Type: "character varying(50)"},
						},
					},
					{
						Name:     "ext_owned",
						SkipDump: true,
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
					},
					{
						Name: "empty_type",
					},
				},
			},
			{
				Name: "geo",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "point2",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "lat", Type: "numeric(9,6)"},
							{Name: "lng", Type: "numeric(9,6)"},
						},
					},
				},
			},
		},
	}

	result, err := GetDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, metadata)
	require.NoError(t, err)

	assert.Contains(t, result, `CREATE TYPE "public"."zz_base" AS (`)
	assert.Contains(t, result, `"street" text COLLATE "C"`)
	assert.Contains(t, result, `"city" character varying(50)`)
	assert.Contains(t, result, `CREATE TYPE "public"."aa_nested" AS (`)
	assert.Contains(t, result, `"home" public.zz_base`)
	assert.Contains(t, result, `"homes" public.zz_base[]`)
	assert.Contains(t, result, `CREATE TYPE "geo"."point2" AS (`)
	assert.Contains(t, result, `CREATE TYPE "public"."empty_type" AS (`)
	assert.Contains(t, result, `COMMENT ON TYPE "public"."zz_base" IS 'base address type';`)
	assert.Contains(t, result, `COMMENT ON COLUMN "public"."zz_base"."street" IS 'street line';`)
	assert.NotContains(t, result, "ext_owned", "skip_dump composite types must not be emitted")

	// Dependency order: zz_base must be created before aa_nested.
	baseIdx := strings.Index(result, `CREATE TYPE "public"."zz_base"`)
	nestedIdx := strings.Index(result, `CREATE TYPE "public"."aa_nested"`)
	require.GreaterOrEqual(t, baseIdx, 0)
	require.GreaterOrEqual(t, nestedIdx, 0)
	assert.Less(t, baseIdx, nestedIdx, "referenced composite must be emitted before its dependent")

	// Enums come before composites.
	enumIdx := strings.Index(result, `CREATE TYPE "public"."order_status"`)
	require.GreaterOrEqual(t, enumIdx, 0)
	assert.Less(t, enumIdx, baseIdx, "enums must be emitted before composite types")

	// The generated SDL must be loadable by the SDL migration engine.
	_, err = schema.DiffSDLMigration(storepb.Engine_POSTGRES, "", result, "")
	require.NoError(t, err, "generated SDL should load: %s", result)
}

func TestGetMultiFileDatabaseDefinition_CompositeTypes(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "mood", Values: []string{"happy", "sad"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "wrapper",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "b", Type: "public.base"},
						},
					},
					{
						Name: "base",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
						Comment: "base type",
					},
				},
			},
		},
	}

	result, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, metadata)
	require.NoError(t, err)

	var typesFile string
	var combined strings.Builder
	for _, file := range result.Files {
		if file.Name == "schemas/public/types.sql" {
			typesFile = file.Content
		}
		combined.WriteString(file.Content)
		combined.WriteString("\n")
	}
	require.NotEmpty(t, typesFile, "types.sql should exist")

	assert.Contains(t, typesFile, `CREATE TYPE "public"."mood" AS ENUM (`)
	assert.Contains(t, typesFile, `CREATE TYPE "public"."base" AS (`)
	assert.Contains(t, typesFile, `CREATE TYPE "public"."wrapper" AS (`)
	assert.Contains(t, typesFile, `COMMENT ON TYPE "public"."base" IS 'base type';`)
	assert.Less(t,
		strings.Index(typesFile, `CREATE TYPE "public"."base" AS (`),
		strings.Index(typesFile, `CREATE TYPE "public"."wrapper" AS (`),
		"referenced composite must be emitted before its dependent")

	_, err = schema.DiffSDLMigration(storepb.Engine_POSTGRES, "", combined.String(), "")
	require.NoError(t, err, "combined multi-file SDL should load")
}

func TestParseQualifiedTypeIdent(t *testing.T) {
	testCases := []struct {
		input  string
		schema string
		name   string
		ok     bool
	}{
		{"integer", "", "", false},
		{"character varying(50)", "", "", false},
		{"numeric(9,6)", "", "", false},
		{"text[]", "", "", false},
		{"timestamp(6) with time zone", "", "", false},
		{"public.addr", "public", "addr", true},
		{"public.addr[]", "public", "addr", true},
		{"public.addr(5)", "public", "addr", true},
		{`"My Schema"."Weird Type"`, "My Schema", "Weird Type", true},
		{`"has""quote".t`, `has"quote`, "t", true},
		{`other.geo`, "other", "geo", true},
	}
	for _, tc := range testCases {
		schemaName, typeName, ok := parseQualifiedTypeIdent(tc.input)
		assert.Equal(t, tc.ok, ok, "input %q", tc.input)
		assert.Equal(t, tc.schema, schemaName, "input %q", tc.input)
		assert.Equal(t, tc.name, typeName, "input %q", tc.input)
	}
}

func TestSortCompositeTypesTopologically(t *testing.T) {
	// Chain across schemas: public.a3 -> other.b2 -> public.c1, with names
	// chosen so alphabetical order is the reverse of dependency order.
	c1 := &storepb.CompositeTypeMetadata{
		Name:       "c1",
		Attributes: []*storepb.CompositeTypeAttribute{{Name: "x", Type: "integer"}},
	}
	b2 := &storepb.CompositeTypeMetadata{
		Name:       "b2",
		Attributes: []*storepb.CompositeTypeAttribute{{Name: "c", Type: "public.c1"}},
	}
	a3 := &storepb.CompositeTypeMetadata{
		Name:       "a3",
		Attributes: []*storepb.CompositeTypeAttribute{{Name: "b", Type: "other.b2[]"}},
	}
	ordered := sortCompositeTypesTopologically([]qualifiedCompositeType{
		{Schema: "public", Composite: a3},
		{Schema: "other", Composite: b2},
		{Schema: "public", Composite: c1},
	})
	require.Len(t, ordered, 3)
	assert.Equal(t, "c1", ordered[0].Composite.Name)
	assert.Equal(t, "b2", ordered[1].Composite.Name)
	assert.Equal(t, "a3", ordered[2].Composite.Name)

	// Independent types come out in deterministic (schema, name) order.
	ordered = sortCompositeTypesTopologically([]qualifiedCompositeType{
		{Schema: "public", Composite: &storepb.CompositeTypeMetadata{Name: "zz"}},
		{Schema: "public", Composite: &storepb.CompositeTypeMetadata{Name: "aa"}},
	})
	require.Len(t, ordered, 2)
	assert.Equal(t, "aa", ordered[0].Composite.Name)
	assert.Equal(t, "zz", ordered[1].Composite.Name)
}
