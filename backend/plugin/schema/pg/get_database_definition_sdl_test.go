package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinitionSDLFormat(t *testing.T) {
	tests := []struct {
		name     string
		metadata *storepb.DatabaseSchemaMetadata
		expected string
	}{
		{
			name: "Simple table with basic columns",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
									{
										Name:     "email",
										Type:     "VARCHAR(320)",
										Nullable: true,
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "email" VARCHAR(320)
);

`,
		},
		{
			name: "Table with default values",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "products",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
									{
										Name:     "price",
										Type:     "DECIMAL(10,2)",
										Default:  "0.00",
										Nullable: false,
									},
									{
										Name:     "active",
										Type:     "BOOLEAN",
										Default:  "true",
										Nullable: false,
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."products" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "price" DECIMAL(10,2) DEFAULT 0.00 NOT NULL,
    "active" BOOLEAN DEFAULT true NOT NULL
);

`,
		},
		{
			name: "Table with constraints",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "email",
										Type:     "VARCHAR(320)",
										Nullable: false,
									},
									{
										Name:     "age",
										Type:     "INTEGER",
										Nullable: true,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
									{
										Name:         "users_email_key",
										Expressions:  []string{"email"},
										Unique:       true,
										IsConstraint: true,
									},
								},
								CheckConstraints: []*storepb.CheckConstraintMetadata{
									{
										Name:       "users_age_check",
										Expression: "(age >= 0)",
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" SERIAL NOT NULL,
    "email" VARCHAR(320) NOT NULL,
    "age" INTEGER,
    CONSTRAINT "users_pkey" PRIMARY KEY (id),
    CONSTRAINT "users_email_key" UNIQUE (email),
    CONSTRAINT "users_age_check" CHECK (age >= 0)
);

`,
		},
		{
			name: "Table with foreign key",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "orders",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "user_id",
										Type:     "INTEGER",
										Nullable: false,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "orders_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
								ForeignKeys: []*storepb.ForeignKeyMetadata{
									{
										Name:              "orders_user_id_fkey",
										Columns:           []string{"user_id"},
										ReferencedSchema:  "public",
										ReferencedTable:   "users",
										ReferencedColumns: []string{"id"},
										OnDelete:          "CASCADE",
										OnUpdate:          "NO ACTION",
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."orders" (
    "id" SERIAL NOT NULL,
    "user_id" INTEGER NOT NULL,
    CONSTRAINT "orders_pkey" PRIMARY KEY (id),
    CONSTRAINT "orders_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON DELETE CASCADE
);

`,
		},
		{
			name: "Multiple tables",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "categories",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(100)",
										Nullable: false,
									},
								},
							},
							{
								Name: "products",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "category_id",
										Type:     "INTEGER",
										Nullable: true,
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."categories" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(100) NOT NULL
);

CREATE TABLE "public"."products" (
    "id" SERIAL NOT NULL,
    "category_id" INTEGER
);

`,
		},
		{
			name: "Table with indexes",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "products",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
									{
										Name:     "category_id",
										Type:     "INTEGER",
										Nullable: true,
									},
									{
										Name:     "price",
										Type:     "DECIMAL(10,2)",
										Nullable: false,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "products_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
									{
										Name:        "idx_products_name",
										Expressions: []string{"name"},
									},
									{
										Name:        "idx_products_category_price",
										Expressions: []string{"category_id", "price"},
										Descending:  []bool{false, true}, // price DESC
									},
									{
										Name:         "idx_products_name_unique",
										Expressions:  []string{"name"},
										Unique:       true,
										IsConstraint: false, // This is a unique index, not a unique constraint
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."products" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "category_id" INTEGER,
    "price" DECIMAL(10,2) NOT NULL,
    CONSTRAINT "products_pkey" PRIMARY KEY (id)
);

CREATE INDEX "idx_products_name" ON ONLY "public"."products" (name);

CREATE INDEX "idx_products_category_price" ON ONLY "public"."products" (category_id, price DESC);

CREATE UNIQUE INDEX "idx_products_name_unique" ON ONLY "public"."products" (name);

`,
		},
		{
			name: "Table with views",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
									{
										Name:     "email",
										Type:     "VARCHAR(320)",
										Nullable: false,
									},
									{
										Name:     "active",
										Type:     "BOOLEAN",
										Default:  "true",
										Nullable: false,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
							},
							{
								Name: "orders",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "user_id",
										Type:     "INTEGER",
										Nullable: false,
									},
									{
										Name:     "total",
										Type:     "DECIMAL(10,2)",
										Nullable: false,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "orders_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
							},
						},
						Views: []*storepb.ViewMetadata{
							{
								Name: "active_users",
								Definition: `SELECT id, name, email
    FROM users
    WHERE active = true`,
							},
							{
								Name: "user_order_summary",
								Definition: `SELECT
    u.id,
    u.name,
    COUNT(o.id) as order_count,
    COALESCE(SUM(o.total), 0) as total_amount
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.name`,
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "email" VARCHAR(320) NOT NULL,
    "active" BOOLEAN DEFAULT true NOT NULL,
    CONSTRAINT "users_pkey" PRIMARY KEY (id)
);

CREATE TABLE "public"."orders" (
    "id" SERIAL NOT NULL,
    "user_id" INTEGER NOT NULL,
    "total" DECIMAL(10,2) NOT NULL,
    CONSTRAINT "orders_pkey" PRIMARY KEY (id)
);

CREATE VIEW "public"."active_users" AS SELECT id, name, email
    FROM users
    WHERE active = true;

CREATE VIEW "public"."user_order_summary" AS SELECT
    u.id,
    u.name,
    COUNT(o.id) as order_count,
    COALESCE(SUM(o.total), 0) as total_amount
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.name;

`,
		},
		{
			name: "Database with functions and procedures",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
									{
										Name:     "created_at",
										Type:     "TIMESTAMP",
										Default:  "CURRENT_TIMESTAMP",
										Nullable: false,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
							},
						},
						Functions: []*storepb.FunctionMetadata{
							{
								Name: "get_user_count",
								Definition: `CREATE FUNCTION "public"."get_user_count"() RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT COUNT(*)::integer FROM users;
$$`,
							},
							{
								Name: "get_user_by_id",
								Definition: `CREATE FUNCTION "public"."get_user_by_id"(user_id integer) RETURNS TABLE(id integer, name character varying, created_at timestamp without time zone)
    LANGUAGE sql
    AS $$
    SELECT u.id, u.name, u.created_at
    FROM users u
    WHERE u.id = user_id;
$$`,
							},
							{
								Name: "update_user_name",
								Definition: `CREATE PROCEDURE "public"."update_user_name"(IN user_id integer, IN new_name character varying)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users
    SET name = new_name
    WHERE id = user_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'User with id % not found', user_id;
    END IF;
END;
$$`,
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT "users_pkey" PRIMARY KEY (id)
);

CREATE FUNCTION "public"."get_user_count"() RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT COUNT(*)::integer FROM users;
$$;

CREATE FUNCTION "public"."get_user_by_id"(user_id integer) RETURNS TABLE(id integer, name character varying, created_at timestamp without time zone)
    LANGUAGE sql
    AS $$
    SELECT u.id, u.name, u.created_at
    FROM users u
    WHERE u.id = user_id;
$$;

CREATE PROCEDURE "public"."update_user_name"(IN user_id integer, IN new_name character varying)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users
    SET name = new_name
    WHERE id = user_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'User with id % not found', user_id;
    END IF;
END;
$$;

`,
		},
		{
			name: "Database with sequences",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*storepb.SequenceMetadata{
							{
								Name:       "independent_seq",
								DataType:   "bigint",
								Start:      "1",
								Increment:  "1",
								MinValue:   "1",
								MaxValue:   "9223372036854775807",
								Cycle:      false,
								OwnerTable: "", // Independent sequence (not owned by any table)
							},
							{
								Name:        "user_id_seq",
								DataType:    "bigint",
								Start:       "1",
								Increment:   "1",
								MinValue:    "1",
								MaxValue:    "9223372036854775807",
								Cycle:       false,
								OwnerTable:  "users",
								OwnerColumn: "id",
							},
							{
								Name:       "order_seq",
								DataType:   "integer",
								Start:      "1000",
								Increment:  "10",
								MinValue:   "1000",
								MaxValue:   "999999",
								Cycle:      true,
								OwnerTable: "", // Independent sequence
							},
						},
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "INTEGER",
										Nullable: false,
										Default:  "nextval('user_id_seq'::regclass)",
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE SEQUENCE "public"."independent_seq" AS bigint START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 NO CYCLE;

CREATE SEQUENCE "public"."order_seq" AS integer START WITH 1000 INCREMENT BY 10 MINVALUE 1000 MAXVALUE 999999 CYCLE;

CREATE TABLE "public"."users" (
    "id" serial,
    "name" VARCHAR(255) NOT NULL
);

`,
		},
		{
			name: "Empty database",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{},
			},
			expected: "",
		},
		{
			name: "Serial columns should use serial types",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*storepb.SequenceMetadata{
							{
								Name:        "users_id_seq",
								DataType:    "bigint",
								Start:       "1",
								Increment:   "1",
								MinValue:    "1",
								MaxValue:    "9223372036854775807",
								Cycle:       false,
								OwnerTable:  "users",
								OwnerColumn: "id",
							},
							{
								Name:        "products_id_seq",
								DataType:    "integer",
								Start:       "1",
								Increment:   "1",
								MinValue:    "1",
								MaxValue:    "2147483647",
								Cycle:       false,
								OwnerTable:  "products",
								OwnerColumn: "id",
							},
							{
								Name:        "orders_id_seq",
								DataType:    "smallint",
								Start:       "1",
								Increment:   "1",
								MinValue:    "1",
								MaxValue:    "32767",
								Cycle:       false,
								OwnerTable:  "orders",
								OwnerColumn: "id",
							},
						},
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "bigint",
										Nullable: false,
										Default:  "nextval('users_id_seq'::regclass)",
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
								},
							},
							{
								Name: "products",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "integer",
										Nullable: false,
										Default:  "nextval('products_id_seq'::regclass)",
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
								},
							},
							{
								Name: "orders",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "smallint",
										Nullable: false,
										Default:  "nextval('orders_id_seq'::regclass)",
									},
									{
										Name:     "user_id",
										Type:     "INTEGER",
										Nullable: false,
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" bigserial,
    "name" VARCHAR(255) NOT NULL
);

CREATE TABLE "public"."products" (
    "id" serial,
    "name" VARCHAR(255) NOT NULL
);

CREATE TABLE "public"."orders" (
    "id" smallserial,
    "user_id" INTEGER NOT NULL
);

`,
		},
		{
			name: "Identity columns should use GENERATED AS IDENTITY syntax",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*storepb.SequenceMetadata{
							{
								Name:        "users_id_seq",
								DataType:    "bigint",
								Start:       "1",
								Increment:   "1",
								MinValue:    "1",
								MaxValue:    "9223372036854775807",
								Cycle:       false,
								OwnerTable:  "users",
								OwnerColumn: "id",
							},
							{
								Name:        "products_id_seq",
								DataType:    "integer",
								Start:       "100",
								Increment:   "5",
								MinValue:    "1",
								MaxValue:    "2147483647",
								Cycle:       false,
								OwnerTable:  "products",
								OwnerColumn: "id",
							},
						},
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:               "id",
										Type:               "bigint",
										Nullable:           false,
										IdentityGeneration: storepb.ColumnMetadata_ALWAYS,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
								},
							},
							{
								Name: "products",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:               "id",
										Type:               "integer",
										Nullable:           false,
										IdentityGeneration: storepb.ColumnMetadata_BY_DEFAULT,
									},
									{
										Name:     "name",
										Type:     "VARCHAR(255)",
										Nullable: false,
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" bigint GENERATED ALWAYS AS IDENTITY,
    "name" VARCHAR(255) NOT NULL
);

CREATE TABLE "public"."products" (
    "id" integer GENERATED BY DEFAULT AS IDENTITY (START WITH 100 INCREMENT BY 5),
    "name" VARCHAR(255) NOT NULL
);

`,
		},
		{
			name: "Table with indexes using custom opclass",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "documents",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "SERIAL",
										Nullable: false,
									},
									{
										Name:     "title",
										Type:     "TEXT",
										Nullable: false,
									},
									{
										Name:     "content",
										Type:     "TEXT",
										Nullable: false,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "documents_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
									{
										Name:            "idx_documents_title_pattern",
										Expressions:     []string{"title"},
										Type:            "btree",
										OpclassNames:    []string{"text_pattern_ops"},
										OpclassDefaults: []bool{false},
									},
									{
										Name:            "idx_documents_title_content",
										Expressions:     []string{"title", "content"},
										Type:            "btree",
										OpclassNames:    []string{"text_pattern_ops", "text_pattern_ops"},
										OpclassDefaults: []bool{false, false},
									},
									{
										Name:            "idx_documents_default_opclass",
										Expressions:     []string{"title"},
										Type:            "btree",
										OpclassNames:    []string{"text_ops"},
										OpclassDefaults: []bool{true}, // Default opclass should not be printed
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."documents" (
    "id" SERIAL NOT NULL,
    "title" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    CONSTRAINT "documents_pkey" PRIMARY KEY (id)
);

CREATE INDEX "idx_documents_title_pattern" ON ONLY "public"."documents" (title text_pattern_ops);

CREATE INDEX "idx_documents_title_content" ON ONLY "public"."documents" (title text_pattern_ops, content text_pattern_ops);

CREATE INDEX "idx_documents_default_opclass" ON ONLY "public"."documents" (title);

`,
		},
		{
			name: "Sequence with ownership (non-serial, non-identity)",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*storepb.SequenceMetadata{
							{
								Name:        "custom_seq",
								DataType:    "bigint",
								Start:       "100",
								Increment:   "5",
								MinValue:    "100",
								MaxValue:    "9223372036854775807",
								Cycle:       false,
								CacheSize:   "10",
								OwnerTable:  "orders",
								OwnerColumn: "order_number",
							},
						},
						Tables: []*storepb.TableMetadata{
							{
								Name: "orders",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "INTEGER",
										Nullable: false,
									},
									{
										Name:     "order_number",
										Type:     "BIGINT",
										Nullable: false,
										Default:  "nextval('custom_seq'::regclass)",
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE SEQUENCE "public"."custom_seq" AS bigint START WITH 100 INCREMENT BY 5 MINVALUE 100 MAXVALUE 9223372036854775807 NO CYCLE CACHE 10;

CREATE TABLE "public"."orders" (
    "id" INTEGER NOT NULL,
    "order_number" BIGINT DEFAULT nextval('custom_seq'::regclass) NOT NULL
);

ALTER SEQUENCE "public"."custom_seq" OWNED BY "public"."orders"."order_number";

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Skip this test case temporarily - will be fixed in a future PR
			// to support ALTER SEQUENCE START WITH for serial columns
			if tt.name == "Sequence with ownership (non-serial, non-identity)" {
				t.Skip("Skipping test case - will support ALTER SEQUENCE START WITH for serial columns in future PR")
			}

			ctx := schema.GetDefinitionContext{
				SDLFormat: true,
			}

			result, err := GetDatabaseDefinition(ctx, tt.metadata)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSchemaSDLDefinition(t *testing.T) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "public",
		Tables: []*storepb.TableMetadata{
			{
				Name: "users",
				Columns: []*storepb.ColumnMetadata{
					{
						Name:     "id",
						Type:     "SERIAL",
						Nullable: false,
					},
					{
						Name:     "name",
						Type:     "VARCHAR(255)",
						Nullable: false,
					},
				},
			},
		},
	}

	result, err := GetSchemaSDLDefinition(schemaMetadata)
	require.NoError(t, err)

	expected := `CREATE TABLE "public"."users" (
    "id" SERIAL NOT NULL,
    "name" VARCHAR(255) NOT NULL
);

`

	assert.Equal(t, expected, result)
}

func TestGetDatabaseDefinitionNormalVsSDLFormat(t *testing.T) {
	// Create a more complex metadata to show the difference
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "SERIAL",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "VARCHAR(255)",
								Nullable: false,
							},
							{
								Name:     "email",
								Type:     "VARCHAR(320)",
								Nullable: true,
							},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        "users_pkey",
								Expressions: []string{"id"},
								Primary:     true,
							},
							{
								Name:        "idx_users_name",
								Expressions: []string{"name"},
							},
							{
								Name:         "idx_users_email_unique",
								Expressions:  []string{"email"},
								Unique:       true,
								IsConstraint: false, // This is a unique index, not a constraint
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "active_users_view",
						Definition: "SELECT id, name, email FROM users WHERE active = true",
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name: "count_active_users",
						Definition: `CREATE FUNCTION "public"."count_active_users"() RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT COUNT(*)::integer FROM users WHERE active = true;
$$`,
					},
				},
			},
		},
	}

	// Test SDL format is false (normal format)
	ctxNormal := schema.GetDefinitionContext{
		SDLFormat: false,
	}

	resultNormal, err := GetDatabaseDefinition(ctxNormal, metadata)
	require.NoError(t, err)
	assert.NotEqual(t, "", resultNormal)
	assert.Contains(t, resultNormal, "CREATE TABLE")

	// Test SDL format is true
	ctxSDL := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	resultSDL, err := GetDatabaseDefinition(ctxSDL, metadata)
	require.NoError(t, err)
	assert.NotEqual(t, "", resultSDL)
	assert.Contains(t, resultSDL, "CREATE TABLE")

	// SDL format should be different from normal format
	// Normal format has separate ALTER TABLE statements for constraints and separate CREATE INDEX
	// SDL format includes constraints within CREATE TABLE and indexes immediately after
	assert.Contains(t, resultNormal, "ALTER TABLE")      // Normal format has separate constraint statements
	assert.NotContains(t, resultSDL, "ALTER TABLE")      // SDL format should not have separate constraints
	assert.Contains(t, resultSDL, "CONSTRAINT")          // SDL format should have inline constraints
	assert.Contains(t, resultSDL, "PRIMARY KEY")         // SDL format should have inline PRIMARY KEY
	assert.Contains(t, resultSDL, "CREATE INDEX")        // SDL format should have CREATE INDEX statements
	assert.Contains(t, resultSDL, "CREATE UNIQUE INDEX") // SDL format should have CREATE UNIQUE INDEX
	assert.Contains(t, resultSDL, "CREATE VIEW")         // SDL format should have CREATE VIEW statements
	assert.Contains(t, resultSDL, "CREATE FUNCTION")     // SDL format should have CREATE FUNCTION statements

	t.Logf("Normal format result: %q", resultNormal)
	t.Logf("SDL format result: %q", resultSDL)
}

// TestCheckConstraintNotValidFormat tests that CHECK constraints with NOT VALID
// are formatted correctly without extra parentheses around the expression
func TestCheckConstraintNotValidFormat(t *testing.T) {
	tests := []struct {
		name     string
		metadata *storepb.DatabaseSchemaMetadata
		expected string
	}{
		{
			name: "Check constraint with NOT VALID",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "namespace_settings",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "namespace_id",
										Type:     "bigint",
										Nullable: false,
									},
									{
										Name:     "default_branch_protection_defaults",
										Type:     "jsonb",
										Nullable: true,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "namespace_settings_pkey",
										Expressions: []string{"namespace_id"},
										Primary:     true,
									},
								},
								CheckConstraints: []*storepb.CheckConstraintMetadata{
									{
										Name:       "default_branch_protection_defaults_size_constraint",
										Expression: "(octet_length(default_branch_protection_defaults::text) <= 1024) NOT VALID",
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."namespace_settings" (
    "namespace_id" bigint NOT NULL,
    "default_branch_protection_defaults" jsonb,
    CONSTRAINT "namespace_settings_pkey" PRIMARY KEY (namespace_id),
    CONSTRAINT "default_branch_protection_defaults_size_constraint" CHECK (octet_length(default_branch_protection_defaults::text) <= 1024) NOT VALID
);

`,
		},
		{
			name: "Multiple check constraints with and without NOT VALID",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "test",
						Tables: []*storepb.TableMetadata{
							{
								Name: "table1",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "serial",
										Nullable: false,
									},
									{
										Name:     "data",
										Type:     "jsonb",
										Nullable: true,
									},
									{
										Name:     "age",
										Type:     "integer",
										Nullable: true,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "table1_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
								CheckConstraints: []*storepb.CheckConstraintMetadata{
									{
										Name:       "table1_data_size_check",
										Expression: "(octet_length(data::text) <= 1024) NOT VALID",
									},
									{
										Name:       "table1_age_check",
										Expression: "(age >= 18)",
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE SCHEMA IF NOT EXISTS "test";

CREATE TABLE "test"."table1" (
    "id" serial NOT NULL,
    "data" jsonb,
    "age" integer,
    CONSTRAINT "table1_pkey" PRIMARY KEY (id),
    CONSTRAINT "table1_data_size_check" CHECK (octet_length(data::text) <= 1024) NOT VALID,
    CONSTRAINT "table1_age_check" CHECK (age >= 18)
);

`,
		},
		{
			name: "Regular check constraint without NOT VALID",
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "users",
								Columns: []*storepb.ColumnMetadata{
									{
										Name:     "id",
										Type:     "serial",
										Nullable: false,
									},
									{
										Name:     "age",
										Type:     "integer",
										Nullable: true,
									},
								},
								Indexes: []*storepb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
								CheckConstraints: []*storepb.CheckConstraintMetadata{
									{
										Name:       "users_age_check",
										Expression: "(age >= 0)",
									},
								},
							},
						},
					},
				},
			},
			expected: `CREATE TABLE "public"."users" (
    "id" serial NOT NULL,
    "age" integer,
    CONSTRAINT "users_pkey" PRIMARY KEY (id),
    CONSTRAINT "users_age_check" CHECK (age >= 0)
);

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := schema.GetDefinitionContext{
				SDLFormat: true,
			}

			result, err := GetDatabaseDefinition(ctx, tt.metadata)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Additional validation: ensure NOT VALID (if present) is outside parentheses
			if strings.Contains(tt.expected, "NOT VALID") {
				// Check that we don't have the incorrect format: CHECK (...) NOT VALID)
				assert.NotContains(t, result, ") NOT VALID)", "NOT VALID should not be inside closing parenthesis")
				// Check that we have the correct format: CHECK (...) NOT VALID
				assert.Contains(t, result, ") NOT VALID", "NOT VALID should be after CHECK expression parenthesis")
			}

			// Validate that the generated SQL can be parsed without errors using ANTLR parser
			_, err = pgparser.ParsePostgreSQL(result)
			require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
		})
	}
}

// TestCheckConstraintNotValidFormatNormalMode tests that CHECK constraints with NOT VALID
// are formatted correctly in normal (non-SDL) mode
func TestCheckConstraintNotValidFormatNormalMode(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "namespace_settings",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "namespace_id",
								Type:     "bigint",
								Nullable: false,
							},
							{
								Name:     "default_branch_protection_defaults",
								Type:     "jsonb",
								Nullable: true,
							},
						},
						CheckConstraints: []*storepb.CheckConstraintMetadata{
							{
								Name:       "default_branch_protection_defaults_size_constraint",
								Expression: "(octet_length(default_branch_protection_defaults::text) <= 1024) NOT VALID",
							},
						},
					},
				},
			},
		},
	}

	// Test normal format (SDLFormat: false)
	ctx := schema.GetDefinitionContext{
		SDLFormat: false,
	}

	result, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	// Verify the CHECK constraint is output correctly in the CREATE TABLE statement
	assert.Contains(t, result, `CONSTRAINT "default_branch_protection_defaults_size_constraint" CHECK (octet_length(default_branch_protection_defaults::text) <= 1024) NOT VALID`)

	// Additional validation: ensure NOT VALID is outside parentheses (not inside extra parentheses)
	assert.NotContains(t, result, ") NOT VALID)", "NOT VALID should not be inside closing parenthesis")
	assert.Contains(t, result, ") NOT VALID", "NOT VALID should be after CHECK expression parenthesis")

	// Validate that the generated SQL can be parsed without errors using ANTLR parser
	_, err = pgparser.ParsePostgreSQL(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetDatabaseDefinitionSDLFormat_WithComments(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name:    "test_schema",
				Comment: "Test schema for comments",
				Tables: []*storepb.TableMetadata{
					{
						Name:    "users",
						Comment: "Users table with comments",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "SERIAL",
								Nullable: false,
								Comment:  "User ID",
							},
							{
								Name:     "name",
								Type:     "VARCHAR(255)",
								Nullable: false,
								Comment:  "User name",
							},
							{
								Name:     "email",
								Type:     "VARCHAR(320)",
								Nullable: true,
								Comment:  "User email address",
							},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        "idx_users_email",
								Expressions: []string{"email"},
								Unique:      false,
								Primary:     false,
								Comment:     "Index on email column",
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "active_users",
						Definition: "SELECT id, name, email FROM test_schema.users WHERE active = true",
						Comment:    "View of active users",
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name:       "get_user_count",
						Signature:  "get_user_count()",
						Definition: "CREATE FUNCTION test_schema.get_user_count() RETURNS INTEGER AS $$ BEGIN RETURN (SELECT COUNT(*) FROM test_schema.users); END; $$ LANGUAGE plpgsql",
						Comment:    "Function to get user count",
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name:      "custom_seq",
						DataType:  "bigint",
						Start:     "1",
						Increment: "1",
						MinValue:  "1",
						MaxValue:  "9223372036854775807",
						Cycle:     false,
						CacheSize: "1",
						Comment:   "Custom sequence for testing",
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	// Verify schema comment
	assert.Contains(t, result, `CREATE SCHEMA IF NOT EXISTS "test_schema";`)
	assert.Contains(t, result, `COMMENT ON SCHEMA "test_schema" IS 'Test schema for comments';`)

	// Verify table comment
	assert.Contains(t, result, `COMMENT ON TABLE "test_schema"."users" IS 'Users table with comments';`)

	// Verify column comments
	assert.Contains(t, result, `COMMENT ON COLUMN "test_schema"."users"."id" IS 'User ID';`)
	assert.Contains(t, result, `COMMENT ON COLUMN "test_schema"."users"."name" IS 'User name';`)
	assert.Contains(t, result, `COMMENT ON COLUMN "test_schema"."users"."email" IS 'User email address';`)

	// Verify view comment
	assert.Contains(t, result, `COMMENT ON VIEW "test_schema"."active_users" IS 'View of active users';`)

	// Verify function comment
	assert.Contains(t, result, `COMMENT ON FUNCTION "test_schema"."get_user_count()" IS 'Function to get user count';`)

	// Verify sequence comment
	assert.Contains(t, result, `COMMENT ON SEQUENCE "test_schema"."custom_seq" IS 'Custom sequence for testing';`)

	// Verify index comment
	assert.Contains(t, result, `COMMENT ON INDEX "test_schema"."idx_users_email" IS 'Index on email column';`)

	// Validate that the generated SQL can be parsed without errors using ANTLR parser
	_, err = pgparser.ParsePostgreSQL(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetDatabaseDefinitionSDLFormat_WithCommentsEscaping(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name:    "test_table",
						Comment: "Table with 'single quotes' and \"double quotes\"",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
								Comment:  "Column with 'quoted' text",
							},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	// Verify that single quotes are properly escaped
	assert.Contains(t, result, `COMMENT ON TABLE "public"."test_table" IS 'Table with ''single quotes'' and "double quotes"';`)
	assert.Contains(t, result, `COMMENT ON COLUMN "public"."test_table"."id" IS 'Column with ''quoted'' text';`)

	// Validate that the generated SQL can be parsed without errors using ANTLR parser
	_, err = pgparser.ParsePostgreSQL(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}
