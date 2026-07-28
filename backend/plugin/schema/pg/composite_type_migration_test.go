package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func newPGDatabaseMetadataWithCompositeTypes(compositeTypes []*storepb.CompositeTypeMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name:           "public",
				CompositeTypes: compositeTypes,
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
}

func TestPGMetadataDiffCreateCompositeTypesInDependencyOrder(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes(nil)
	// aa_nested sorts before zz_base alphabetically but depends on it.
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "aa_nested",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "home", Type: "public.zz_base"},
			},
		},
		{
			Name:    "zz_base",
			Comment: "base type",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "street", Type: "text", Collation: `"C"`, Comment: "street line"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, `CREATE TYPE "public"."zz_base" AS (`)
	require.Contains(t, sql, `"street" text COLLATE "C"`)
	require.Contains(t, sql, `CREATE TYPE "public"."aa_nested" AS (`)
	require.Contains(t, sql, `COMMENT ON TYPE "public"."zz_base" IS 'base type';`)
	require.Contains(t, sql, `COMMENT ON COLUMN "public"."zz_base"."street" IS 'street line';`)
	require.Less(t,
		strings.Index(sql, `CREATE TYPE "public"."zz_base"`),
		strings.Index(sql, `CREATE TYPE "public"."aa_nested"`),
		"referenced composite must be created before its dependent")
}

func TestPGMetadataDiffDropCompositeTypesInReverseDependencyOrder(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "zz_base",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "x", Type: "integer"},
			},
		},
		{
			Name: "aa_nested",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "home", Type: "public.zz_base"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes(nil)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, `DROP TYPE "public"."aa_nested";`)
	require.Contains(t, sql, `DROP TYPE "public"."zz_base";`)
	require.Less(t,
		strings.Index(sql, `DROP TYPE "public"."aa_nested"`),
		strings.Index(sql, `DROP TYPE "public"."zz_base"`),
		"dependent composite must be dropped before the composite it references")
}

func TestPGMetadataDiffCompositeDropsPrecedeEnumDrops(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a", "b"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "uses_status",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "s", Type: "public.status"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := newPGDatabaseMetadataWithCompositeTypes(nil)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Less(t,
		strings.Index(sql, `DROP TYPE "public"."uses_status"`),
		strings.Index(sql, `DROP TYPE "public"."status"`),
		"composite referencing an enum must be dropped before the enum")
}

func TestPGMetadataDiffAlterCompositeTypeAttributes(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "street", Type: "text"},
				{Name: "city", Type: "character varying(50)"},
				{Name: "dropped_attr", Type: "integer"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "street", Type: "text"},
				{Name: "city", Type: "character varying(100)"},
				{Name: "zip", Type: "text", Collation: `"C"`},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, `ALTER TYPE "public"."addr" DROP ATTRIBUTE "dropped_attr";`)
	require.Contains(t, sql, `ALTER TYPE "public"."addr" ADD ATTRIBUTE "zip" text COLLATE "C";`)
	require.Contains(t, sql, `ALTER TYPE "public"."addr" ALTER ATTRIBUTE "city" TYPE character varying(100);`)
	require.NotContains(t, sql, `"street"`, "unchanged attribute must not produce DDL")
}

func TestPGMetadataDiffCompositeTypeReorderOnlyWarns(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "a", Type: "text"},
				{Name: "b", Type: "integer"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "b", Type: "integer"},
				{Name: "a", Type: "text"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, `-- WARNING: PostgreSQL does not support reordering the attributes of "public"."addr"`)
	require.NotContains(t, sql, "ALTER TYPE", "reorder-only change must not produce attribute DDL")
	require.NotContains(t, sql, "DROP TYPE")
}

func TestPGMetadataDiffCompositeTypeCommentChanges(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name:    "addr",
			Comment: "old type comment",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "street", Type: "text", Comment: "old attr comment"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name:    "addr",
			Comment: "new type comment",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "street", Type: "text", Comment: "new attr comment"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, `COMMENT ON TYPE "public"."addr" IS 'new type comment';`)
	require.Contains(t, sql, `COMMENT ON COLUMN "public"."addr"."street" IS 'new attr comment';`)
	require.NotContains(t, sql, "ALTER TYPE", "comment-only change must not produce attribute DDL")
}

func TestPGMetadataDiffIdenticalCompositeTypesProduceNoMigration(t *testing.T) {
	composites := []*storepb.CompositeTypeMetadata{
		{
			Name:    "addr",
			Comment: "c",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "street", Type: "text", Collation: `"C"`, Comment: "s"},
			},
		},
	}
	source := newPGDatabaseMetadataWithCompositeTypes(composites)
	target := newPGDatabaseMetadataWithCompositeTypes(composites)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Empty(t, sql)
}

func TestPGMetadataDiffCompositeDropDeferredAfterColumnRetype(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "public.addr", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	dropIdx := strings.Index(sql, `DROP TYPE "public"."addr"`)
	alterIdx := strings.Index(sql, `ALTER COLUMN "c"`)
	require.GreaterOrEqual(t, dropIdx, 0, "composite type must still be dropped: %s", sql)
	require.GreaterOrEqual(t, alterIdx, 0, "column must be retyped: %s", sql)
	require.Greater(t, dropIdx, alterIdx,
		"composite drop must run after the column retype that releases it: %s", sql)
}

func TestPGMetadataDiffSchemaDropIncludesCompositeTypes(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
			{
				Name: "s",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	dropTypeIdx := strings.Index(sql, `DROP TYPE "s"."addr"`)
	dropSchemaIdx := strings.Index(sql, `DROP SCHEMA`)
	require.GreaterOrEqual(t, dropTypeIdx, 0, "schema drop must drop its composite types first: %s", sql)
	require.GreaterOrEqual(t, dropSchemaIdx, 0)
	require.Less(t, dropTypeIdx, dropSchemaIdx, "composite type drop must precede DROP SCHEMA: %s", sql)
}

func TestPGMetadataDiffCompositeTypeMiddleInsertWarns(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "a", Type: "text"},
				{Name: "b", Type: "text"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "a", Type: "text"},
				{Name: "x", Type: "text"},
				{Name: "b", Type: "text"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Contains(t, sql, `ALTER TYPE "public"."addr" ADD ATTRIBUTE "x" text;`)
	require.Contains(t, sql, `-- WARNING: PostgreSQL does not support reordering the attributes of "public"."addr"`,
		"middle insertion is not achievable via ADD ATTRIBUTE and must warn: %s", sql)
}

func TestPGMetadataDiffDeferredCompositeDefersReferencedEnumDrop(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "public.addr", Nullable: true},
						},
					},
				},
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a", "b"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "s", Type: "public.status"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	retypeIdx := strings.Index(sql, `ALTER COLUMN "c"`)
	compositeIdx := strings.Index(sql, `DROP TYPE "public"."addr"`)
	enumIdx := strings.Index(sql, `DROP TYPE "public"."status"`)
	require.GreaterOrEqual(t, retypeIdx, 0, "column must be retyped: %s", sql)
	require.GreaterOrEqual(t, compositeIdx, 0, "composite must be dropped: %s", sql)
	require.GreaterOrEqual(t, enumIdx, 0, "enum must be dropped: %s", sql)
	require.Greater(t, compositeIdx, retypeIdx, "deferred composite drop must follow the retype: %s", sql)
	require.Greater(t, enumIdx, compositeIdx, "enum referenced by the deferred composite must drop after it: %s", sql)
}

func TestPGMetadataDiffSchemaDropDeferredWhenCompositeReleasedByRetype(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "s.addr", Nullable: true},
						},
					},
				},
			},
			{
				Name: "s",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	retypeIdx := strings.Index(sql, `ALTER COLUMN "c"`)
	dropTypeIdx := strings.Index(sql, `DROP TYPE "s"."addr"`)
	dropSchemaIdx := strings.Index(sql, `DROP SCHEMA IF EXISTS "s"`)
	require.GreaterOrEqual(t, retypeIdx, 0, "column must be retyped: %s", sql)
	require.GreaterOrEqual(t, dropTypeIdx, 0, "composite in dropped schema must be dropped: %s", sql)
	require.GreaterOrEqual(t, dropSchemaIdx, 0, "schema must be dropped: %s", sql)
	require.Greater(t, dropTypeIdx, retypeIdx, "schema-owned composite drop must follow the retype releasing it: %s", sql)
	require.Greater(t, dropSchemaIdx, dropTypeIdx, "DROP SCHEMA must follow its composite drop: %s", sql)
}

func TestPGMetadataDiffSchemaDropDeferredWhenDeferredCompositeReferencesItsEnum(t *testing.T) {
	// public.addr is deferred (public.t.c retyped away from it) and references
	// s.status; schema s (containing only that enum) must defer its drop too.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "public.addr", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "st", Type: "s.status"},
						},
					},
				},
			},
			{
				Name: "s",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a"}},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	compositeIdx := strings.Index(sql, `DROP TYPE "public"."addr"`)
	enumIdx := strings.Index(sql, `DROP TYPE "s"."status"`)
	schemaIdx := strings.Index(sql, `DROP SCHEMA IF EXISTS "s"`)
	require.GreaterOrEqual(t, compositeIdx, 0, "deferred composite must be dropped: %s", sql)
	require.GreaterOrEqual(t, enumIdx, 0, "enum in dropped schema must be dropped: %s", sql)
	require.GreaterOrEqual(t, schemaIdx, 0, "schema must be dropped: %s", sql)
	require.Greater(t, enumIdx, compositeIdx,
		"enum referenced by the deferred composite must drop after it, even inside a schema drop: %s", sql)
}

func TestPGMetadataDiffDeferredSchemaCompositeDefersReferencedEnumDrop(t *testing.T) {
	// Schema s is deferred (public.t.c retyped away from s.addr); s.addr
	// references public.status, a top-level dropped enum, which therefore
	// must also defer until after the schema drop removes s.addr.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "s.addr", Nullable: true},
						},
					},
				},
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a"}},
				},
			},
			{
				Name: "s",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "st", Type: "public.status"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	retypeIdx := strings.Index(sql, `ALTER COLUMN "c"`)
	schemaCompositeIdx := strings.Index(sql, `DROP TYPE "s"."addr"`)
	enumIdx := strings.Index(sql, `DROP TYPE "public"."status"`)
	require.GreaterOrEqual(t, retypeIdx, 0, "column must be retyped: %s", sql)
	require.GreaterOrEqual(t, schemaCompositeIdx, 0, "deferred schema's composite must be dropped: %s", sql)
	require.GreaterOrEqual(t, enumIdx, 0, "enum must be dropped: %s", sql)
	require.Greater(t, schemaCompositeIdx, retypeIdx, "schema-owned composite drop must follow the retype: %s", sql)
	require.Greater(t, enumIdx, schemaCompositeIdx,
		"enum referenced from the deferred schema's composite must drop last: %s", sql)
}

func TestPGMetadataDiffCyclicDeferredSchemaDropsGlobalizeTypeDrops(t *testing.T) {
	// Schemas a and b reference each other: a.ca uses b.eb, b.cb uses a.ea.
	// Both are deferred (a column retype releases a.ca). Globalized type
	// drops must remove both composites before either enum, regardless of
	// the schema-level cycle.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "a.ca", Nullable: true},
						},
					},
				},
			},
			{
				Name: "a",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "ea", Values: []string{"x"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "ca",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "v", Type: "b.eb"},
						},
					},
				},
			},
			{
				Name: "b",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "eb", Values: []string{"y"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "cb",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "v", Type: "a.ea"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	caIdx := strings.Index(sql, `DROP TYPE "a"."ca"`)
	cbIdx := strings.Index(sql, `DROP TYPE "b"."cb"`)
	eaIdx := strings.Index(sql, `DROP TYPE "a"."ea"`)
	ebIdx := strings.Index(sql, `DROP TYPE "b"."eb"`)
	require.GreaterOrEqual(t, caIdx, 0, "a.ca must be dropped: %s", sql)
	require.GreaterOrEqual(t, cbIdx, 0, "b.cb must be dropped: %s", sql)
	require.GreaterOrEqual(t, eaIdx, 0, "a.ea must be dropped: %s", sql)
	require.GreaterOrEqual(t, ebIdx, 0, "b.eb must be dropped: %s", sql)
	require.Greater(t, eaIdx, cbIdx, "a.ea must drop after b.cb which references it: %s", sql)
	require.Greater(t, ebIdx, caIdx, "b.eb must drop after a.ca which references it: %s", sql)
	dropSchemaAIdx := strings.Index(sql, `DROP SCHEMA IF EXISTS "a"`)
	dropSchemaBIdx := strings.Index(sql, `DROP SCHEMA IF EXISTS "b"`)
	require.Greater(t, dropSchemaAIdx, eaIdx, "DROP SCHEMA a must come after its types: %s", sql)
	require.Greater(t, dropSchemaBIdx, ebIdx, "DROP SCHEMA b must come after its types: %s", sql)
}

func TestPGMetadataDiffImmediateSchemaDropOrdersTypesGlobally(t *testing.T) {
	// Dropping schema s (whose composite references top-level types) together
	// with those top-level types: s.child must drop before public.base and
	// public.status regardless of the schema-drop boundary.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "base",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
					},
				},
			},
			{
				Name: "s",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "child",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "b", Type: "public.base"},
							{Name: "st", Type: "public.status"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	childIdx := strings.Index(sql, `DROP TYPE "s"."child"`)
	baseIdx := strings.Index(sql, `DROP TYPE "public"."base"`)
	statusIdx := strings.Index(sql, `DROP TYPE "public"."status"`)
	schemaIdx := strings.Index(sql, `DROP SCHEMA IF EXISTS "s"`)
	require.GreaterOrEqual(t, childIdx, 0, "schema-owned composite must be dropped: %s", sql)
	require.GreaterOrEqual(t, baseIdx, 0)
	require.GreaterOrEqual(t, statusIdx, 0)
	require.GreaterOrEqual(t, schemaIdx, 0)
	require.Less(t, childIdx, baseIdx, "dependent s.child must drop before public.base: %s", sql)
	require.Less(t, childIdx, statusIdx, "dependent s.child must drop before public.status: %s", sql)
	require.Greater(t, schemaIdx, childIdx, "DROP SCHEMA must come after its composite drop: %s", sql)
}

func TestPGMetadataDiffCompositeCreateWaitsForTableRowType(t *testing.T) {
	// Creating table t and composites c1(public.t) and c2(c1): both
	// composites must be created after the table.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c1",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "r", Type: "public.t"},
						},
					},
					{
						Name: "c2",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "w", Type: "public.c1"},
						},
					},
					{
						Name: "plain",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	tableIdx := strings.Index(sql, `CREATE TABLE "public"."t"`)
	c1Idx := strings.Index(sql, `CREATE TYPE "public"."c1"`)
	c2Idx := strings.Index(sql, `CREATE TYPE "public"."c2"`)
	plainIdx := strings.Index(sql, `CREATE TYPE "public"."plain"`)
	require.GreaterOrEqual(t, tableIdx, 0, "table must be created: %s", sql)
	require.GreaterOrEqual(t, c1Idx, 0)
	require.GreaterOrEqual(t, c2Idx, 0)
	require.GreaterOrEqual(t, plainIdx, 0)
	require.Greater(t, c1Idx, tableIdx, "composite using the table row type must be created after the table: %s", sql)
	require.Greater(t, c2Idx, c1Idx, "dependent of a post-table composite must follow it: %s", sql)
	require.Less(t, plainIdx, tableIdx, "independent composite stays before tables: %s", sql)
}

func TestPGMetadataDiffDeferredCompositeDefersItsCompositeDependency(t *testing.T) {
	// child(public.base) is deferred by a column retype; base, also dropped,
	// must defer too so it drops after child.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "public.child", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "base",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
					},
					{
						Name: "child",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "b", Type: "public.base"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	retypeIdx := strings.Index(sql, `ALTER COLUMN "c"`)
	childIdx := strings.Index(sql, `DROP TYPE "public"."child"`)
	baseIdx := strings.Index(sql, `DROP TYPE "public"."base"`)
	require.GreaterOrEqual(t, retypeIdx, 0)
	require.GreaterOrEqual(t, childIdx, 0)
	require.GreaterOrEqual(t, baseIdx, 0)
	require.Greater(t, childIdx, retypeIdx, "deferred child drops after the retype: %s", sql)
	require.Greater(t, baseIdx, childIdx, "base referenced by deferred child must defer and drop after it: %s", sql)
}

func TestPGMetadataDiffRowTypeCompositeDropsBeforeTable(t *testing.T) {
	// Dropping table t and composite c(r public.t): the composite must drop
	// before the table whose row type it references.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "r", Type: "public.t"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	typeIdx := strings.Index(sql, `DROP TYPE "public"."c"`)
	tableIdx := strings.Index(sql, `DROP TABLE "public"."t"`)
	require.GreaterOrEqual(t, typeIdx, 0, "composite must be dropped: %s", sql)
	require.GreaterOrEqual(t, tableIdx, 0, "table must be dropped: %s", sql)
	require.Less(t, typeIdx, tableIdx, "row-type-referencing composite must drop before the table: %s", sql)
}

func TestPGMetadataDiffCompositeAlterWaitsForNewTableRowType(t *testing.T) {
	// Existing composite gains an attribute typed with a row type of a table
	// created in the same migration; the alter must follow CREATE TABLE.
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "c",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "x", Type: "integer"},
			},
		},
	})
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
							{Name: "r", Type: "public.t"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	tableIdx := strings.Index(sql, `CREATE TABLE "public"."t"`)
	alterIdx := strings.Index(sql, `ALTER TYPE "public"."c" ADD ATTRIBUTE "r" public.t;`)
	require.GreaterOrEqual(t, tableIdx, 0, "table must be created: %s", sql)
	require.GreaterOrEqual(t, alterIdx, 0, "attribute must be added: %s", sql)
	require.Greater(t, alterIdx, tableIdx, "composite alter must follow the table creation it references: %s", sql)
}

func TestPGMetadataDiffCreateSandwichTableCompositeTable(t *testing.T) {
	// r (row-type provider) -> c (composite using r) -> holder (table using c):
	// creation order must interleave through the shared graph.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "r",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
					{
						Name: "holder",
						Columns: []*storepb.ColumnMetadata{
							{Name: "x", Type: "public.c", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "row", Type: "public.r"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	rIdx := strings.Index(sql, `CREATE TABLE "public"."r"`)
	cIdx := strings.Index(sql, `CREATE TYPE "public"."c"`)
	holderIdx := strings.Index(sql, `CREATE TABLE "public"."holder"`)
	require.GreaterOrEqual(t, rIdx, 0)
	require.GreaterOrEqual(t, cIdx, 0)
	require.GreaterOrEqual(t, holderIdx, 0)
	require.Greater(t, cIdx, rIdx, "composite must follow its row-type provider table: %s", sql)
	require.Greater(t, holderIdx, cIdx, "consumer table must follow the composite: %s", sql)
}

func TestPGMetadataDiffDropSandwichTableCompositeTable(t *testing.T) {
	// holder (table using c) -> c (composite using r) -> r: drop order must
	// interleave through the shared graph.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "r",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
					{
						Name: "holder",
						Columns: []*storepb.ColumnMetadata{
							{Name: "x", Type: "public.c", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "row", Type: "public.r"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	holderIdx := strings.Index(sql, `DROP TABLE "public"."holder"`)
	cIdx := strings.Index(sql, `DROP TYPE "public"."c"`)
	rIdx := strings.Index(sql, `DROP TABLE "public"."r"`)
	require.GreaterOrEqual(t, holderIdx, 0)
	require.GreaterOrEqual(t, cIdx, 0)
	require.GreaterOrEqual(t, rIdx, 0)
	require.Less(t, holderIdx, cIdx, "table using the composite must drop before it: %s", sql)
	require.Less(t, cIdx, rIdx, "composite must drop before its row-type provider table: %s", sql)
}

func TestPGMetadataDiffDeferredSchemaCompositeDefersTopLevelDependency(t *testing.T) {
	// Schema s is deferred (retype releases s.child); s.child references
	// top-level dropped public.base, which must defer too.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "s.child", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "base",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
					},
				},
			},
			{
				Name: "s",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "child",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "b", Type: "public.base"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "c", Type: "text", Nullable: true},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	retypeIdx := strings.Index(sql, `ALTER COLUMN "c"`)
	childIdx := strings.Index(sql, `DROP TYPE "s"."child"`)
	baseIdx := strings.Index(sql, `DROP TYPE "public"."base"`)
	require.GreaterOrEqual(t, retypeIdx, 0)
	require.GreaterOrEqual(t, childIdx, 0)
	require.GreaterOrEqual(t, baseIdx, 0)
	require.Greater(t, childIdx, retypeIdx, "deferred schema composite drops after the retype: %s", sql)
	require.Greater(t, baseIdx, childIdx, "top-level base referenced by the deferred schema composite must defer and drop after it: %s", sql)
}

func TestPGMetadataDiffCompositeDropDeferredAfterColumnDrop(t *testing.T) {
	// Dropping the only column using a composite (on a surviving table)
	// releases it in writeDropAlterTableObjects; the composite drop must
	// defer past that.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "c", Type: "public.addr", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	columnDropIdx := strings.Index(sql, `DROP COLUMN "c"`)
	typeDropIdx := strings.Index(sql, `DROP TYPE "public"."addr"`)
	require.GreaterOrEqual(t, columnDropIdx, 0, "column must be dropped: %s", sql)
	require.GreaterOrEqual(t, typeDropIdx, 0, "composite must be dropped: %s", sql)
	require.Greater(t, typeDropIdx, columnDropIdx,
		"composite drop must follow the column drop that releases it: %s", sql)
}

func TestPGMetadataDiffReleasingCompositeAlterPrecedesTableDrop(t *testing.T) {
	// Retyping c.r away from public.t's row type releases the dropped table;
	// the releasing ALTER must precede DROP TABLE.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "r", Type: "public.t"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "r", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	alterIdx := strings.Index(sql, `ALTER TYPE "public"."c" ALTER ATTRIBUTE "r" TYPE text;`)
	dropIdx := strings.Index(sql, `DROP TABLE "public"."t"`)
	require.GreaterOrEqual(t, alterIdx, 0, "releasing alter must be emitted: %s", sql)
	require.GreaterOrEqual(t, dropIdx, 0, "table must be dropped: %s", sql)
	require.Less(t, alterIdx, dropIdx, "releasing alter must precede the table drop: %s", sql)
	require.Equal(t, strings.Count(sql, `ALTER ATTRIBUTE "r"`), 1, "releasing alter must not be emitted twice: %s", sql)
}

func TestPGMetadataDiffCompositeArrayColumnOrdersCreate(t *testing.T) {
	// Column sync renders composite-array columns as the bare "_name" form;
	// the create graph must still order the composite first. zz table name
	// sorts after the composite to defeat accidental ordering.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "aa_holder",
						Columns: []*storepb.ColumnMetadata{
							{Name: "xs", Type: "_zz_addr", Nullable: true},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "zz_addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	typeIdx := strings.Index(sql, `CREATE TYPE "public"."zz_addr"`)
	tableIdx := strings.Index(sql, `CREATE TABLE "public"."aa_holder"`)
	require.GreaterOrEqual(t, typeIdx, 0)
	require.GreaterOrEqual(t, tableIdx, 0)
	require.Less(t, typeIdx, tableIdx, "array-typed column must order the composite before the table: %s", sql)
}

func TestPGMetadataDiffSchemaDropEmitsCompositeDropOnce(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
			{
				Name: "s",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(sql, `DROP TYPE "s"."addr";`),
		"schema-owned composite must drop exactly once: %s", sql)
}

func TestPGMetadataDiffReleasingRetypeToCreatedTableSplitsDropReadd(t *testing.T) {
	// c.r moves from dropped old_t's row type to created new_t's row type:
	// the drop phase removes the attribute, the create phase re-adds it
	// after the new table exists.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "old_t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "r", Type: "public.old_t"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "new_t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "r", Type: "public.new_t"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	dropAttrIdx := strings.Index(sql, `ALTER TYPE "public"."c" DROP ATTRIBUTE "r";`)
	dropTableIdx := strings.Index(sql, `DROP TABLE "public"."old_t"`)
	createTableIdx := strings.Index(sql, `CREATE TABLE "public"."new_t"`)
	addAttrIdx := strings.Index(sql, `ALTER TYPE "public"."c" ADD ATTRIBUTE "r" public.new_t;`)
	require.GreaterOrEqual(t, dropAttrIdx, 0, "releasing drop must be emitted: %s", sql)
	require.GreaterOrEqual(t, dropTableIdx, 0)
	require.GreaterOrEqual(t, createTableIdx, 0)
	require.GreaterOrEqual(t, addAttrIdx, 0, "attribute must be re-added: %s", sql)
	require.Less(t, dropAttrIdx, dropTableIdx, "attribute drop must precede the table drop: %s", sql)
	require.Greater(t, addAttrIdx, createTableIdx, "re-add must follow the new table: %s", sql)
	require.NotContains(t, sql, `ALTER ATTRIBUTE "r"`, "no premature retype may be emitted: %s", sql)
}

func TestPGMetadataDiffCompositeCreateWaitsForViewRowType(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Views: []*storepb.ViewMetadata{
					{Name: "v", Definition: "SELECT 1 AS a"},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "row", Type: "public.v"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	viewIdx := strings.Index(sql, `VIEW "public"."v"`)
	typeIdx := strings.Index(sql, `CREATE TYPE "public"."c"`)
	require.GreaterOrEqual(t, viewIdx, 0, "view must be created: %s", sql)
	require.GreaterOrEqual(t, typeIdx, 0, "composite must be created: %s", sql)
	require.Greater(t, typeIdx, viewIdx, "composite using a view row type must follow the view: %s", sql)
}

func TestPGMetadataDiffCompositeDropReleasedByCompositeAlterDefers(t *testing.T) {
	// parent's alter (child -> text) releases dropped child only in the
	// create phase, so child's drop must defer past it.
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "child",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "x", Type: "integer"},
			},
		},
		{
			Name: "parent",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "c", Type: "public.child"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "parent",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "c", Type: "text"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	alterIdx := strings.Index(sql, `ALTER TYPE "public"."parent" ALTER ATTRIBUTE "c" TYPE text;`)
	dropIdx := strings.Index(sql, `DROP TYPE "public"."child"`)
	require.GreaterOrEqual(t, alterIdx, 0, "releasing alter must be emitted: %s", sql)
	require.GreaterOrEqual(t, dropIdx, 0, "child must be dropped: %s", sql)
	require.Greater(t, dropIdx, alterIdx, "child drop must follow the alter that releases it: %s", sql)
}

func TestPGMetadataDiffEnumDropReleasedByCompositeAlterDefers(t *testing.T) {
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "s", Type: "public.status"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "addr",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "s", Type: "text"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	alterIdx := strings.Index(sql, `ALTER TYPE "public"."addr" ALTER ATTRIBUTE "s" TYPE text;`)
	dropIdx := strings.Index(sql, `DROP TYPE "public"."status"`)
	require.GreaterOrEqual(t, alterIdx, 0, "releasing alter must be emitted: %s", sql)
	require.GreaterOrEqual(t, dropIdx, 0, "enum must be dropped: %s", sql)
	require.Greater(t, dropIdx, alterIdx, "enum drop must follow the alter that releases it: %s", sql)
}

func TestPGMetadataDiffReleasingAlterPrecedesViewDrop(t *testing.T) {
	// c.row moves off dropped view v's row type; the early alter must
	// precede DROP VIEW.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Views: []*storepb.ViewMetadata{
					{Name: "v", Definition: "SELECT 1 AS a"},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "row", Type: "public.v"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "c",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "row", Type: "text"},
			},
		},
	})

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	alterIdx := strings.Index(sql, `ALTER TYPE "public"."c" ALTER ATTRIBUTE "row" TYPE text;`)
	dropViewIdx := strings.Index(sql, `DROP VIEW "public"."v"`)
	require.GreaterOrEqual(t, alterIdx, 0, "releasing alter must be emitted: %s", sql)
	require.GreaterOrEqual(t, dropViewIdx, 0, "view must be dropped: %s", sql)
	require.Less(t, alterIdx, dropViewIdx, "releasing alter must precede the view drop: %s", sql)
}

func TestPGMetadataDiffCompositeAlterWaitsForCreatedViewRowType(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name: "c",
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "x", Type: "integer"},
			},
		},
	})
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Views: []*storepb.ViewMetadata{
					{Name: "v", Definition: "SELECT 1 AS a"},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
							{Name: "row", Type: "public.v"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	viewIdx := strings.Index(sql, `VIEW "public"."v"`)
	alterIdx := strings.Index(sql, `ALTER TYPE "public"."c" ADD ATTRIBUTE "row" public.v;`)
	require.GreaterOrEqual(t, viewIdx, 0, "view must be created: %s", sql)
	require.GreaterOrEqual(t, alterIdx, 0, "attribute must be added: %s", sql)
	require.Greater(t, alterIdx, viewIdx, "alter gaining the view row type must follow the view: %s", sql)
}

func TestPGMetadataDiffCompositeDropPrecedesReferencedViewDrop(t *testing.T) {
	// Dropping both c(row public.v) and v: the composite must drop first
	// despite the blanket views-before-composites edges.
	source := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Views: []*storepb.ViewMetadata{
					{Name: "v", Definition: "SELECT 1 AS a"},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "c",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "row", Type: "public.v"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
	target := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}, nil, nil, storepb.Engine_POSTGRES, true)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	typeIdx := strings.Index(sql, `DROP TYPE "public"."c"`)
	viewIdx := strings.Index(sql, `DROP VIEW "public"."v"`)
	require.GreaterOrEqual(t, typeIdx, 0, "composite must be dropped: %s", sql)
	require.GreaterOrEqual(t, viewIdx, 0, "view must be dropped: %s", sql)
	require.Less(t, typeIdx, viewIdx, "composite must drop before the view whose row type it references: %s", sql)
}

func TestFilterPostgresArchiveSchemaKeepsCompositeTypes(t *testing.T) {
	diff := &schema.MetadataDiff{
		CompositeTypeChanges: []*schema.CompositeTypeDiff{
			{Action: schema.MetadataDiffActionCreate, SchemaName: "public", CompositeTypeName: "addr"},
			{Action: schema.MetadataDiffActionCreate, SchemaName: "bbdataarchive", CompositeTypeName: "archived"},
		},
	}

	filtered := schema.FilterPostgresArchiveSchema(diff)

	require.Len(t, filtered.CompositeTypeChanges, 1)
	require.Equal(t, "public", filtered.CompositeTypeChanges[0].SchemaName)
}

func TestPGMetadataDiffSkipDumpCompositeTypesIgnored(t *testing.T) {
	source := newPGDatabaseMetadataWithCompositeTypes([]*storepb.CompositeTypeMetadata{
		{
			Name:     "ext_owned",
			SkipDump: true,
			Attributes: []*storepb.CompositeTypeAttribute{
				{Name: "x", Type: "integer"},
			},
		},
	})
	target := newPGDatabaseMetadataWithCompositeTypes(nil)

	sql, err := schema.DiffMigration(storepb.Engine_POSTGRES, source, target)

	require.NoError(t, err)
	require.Empty(t, sql, "skip_dump composite types must not be diffed")
}
