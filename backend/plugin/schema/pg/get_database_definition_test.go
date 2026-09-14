package pg

import (
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"os"

	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetDatabaseDefinitionSDLFormat(t *testing.T) {
	tests := []struct {
		name     string
		metadata *metadatapb.DatabaseSchemaMetadata
		expected string
	}{
		{
			name: "Simple table with basic columns",
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "products",
								Columns: []*metadatapb.ColumnMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
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
								CheckConstraints: []*metadatapb.CheckConstraintMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "orders",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "orders_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
								ForeignKeys: []*metadatapb.ForeignKeyMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "categories",
								Columns: []*metadatapb.ColumnMetadata{
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
								Columns: []*metadatapb.ColumnMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "products",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
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

CREATE INDEX "idx_products_name" ON "public"."products" (name);

CREATE INDEX "idx_products_category_price" ON "public"."products" (category_id, price DESC);

CREATE UNIQUE INDEX "idx_products_name_unique" ON "public"."products" (name);

`,
		},
		{
			name: "Table with views",
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
							},
							{
								Name: "orders",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "orders_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
							},
						},
						Views: []*metadatapb.ViewMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
							},
						},
						Functions: []*metadatapb.FunctionMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*metadatapb.SequenceMetadata{
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
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{},
			},
			expected: "",
		},
		{
			name: "Serial columns should use serial types",
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*metadatapb.SequenceMetadata{
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
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
								Columns: []*metadatapb.ColumnMetadata{
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
								Columns: []*metadatapb.ColumnMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*metadatapb.SequenceMetadata{
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
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
									{
										Name:               "id",
										Type:               "bigint",
										Nullable:           false,
										IdentityGeneration: metadatapb.ColumnMetadata_ALWAYS,
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
								Columns: []*metadatapb.ColumnMetadata{
									{
										Name:               "id",
										Type:               "integer",
										Nullable:           false,
										IdentityGeneration: metadatapb.ColumnMetadata_BY_DEFAULT,
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "documents",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
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

CREATE INDEX "idx_documents_title_pattern" ON "public"."documents" (title text_pattern_ops);

CREATE INDEX "idx_documents_title_content" ON "public"."documents" (title text_pattern_ops, content text_pattern_ops);

CREATE INDEX "idx_documents_default_opclass" ON "public"."documents" (title);

`,
		},
		{
			name: "Sequence with ownership (non-serial, non-identity)",
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Sequences: []*metadatapb.SequenceMetadata{
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
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "orders",
								Columns: []*metadatapb.ColumnMetadata{
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
	schemaMetadata := &metadatapb.SchemaMetadata{
		Name: "public",
		Tables: []*metadatapb.TableMetadata{
			{
				Name: "users",
				Columns: []*metadatapb.ColumnMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
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
						Indexes: []*metadatapb.IndexMetadata{
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
				Views: []*metadatapb.ViewMetadata{
					{
						Name:       "active_users_view",
						Definition: "SELECT id, name, email FROM users WHERE active = true",
					},
				},
				Functions: []*metadatapb.FunctionMetadata{
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
		metadata *metadatapb.DatabaseSchemaMetadata
		expected string
	}{
		{
			name: "Check constraint with NOT VALID",
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "namespace_settings",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "namespace_settings_pkey",
										Expressions: []string{"namespace_id"},
										Primary:     true,
									},
								},
								CheckConstraints: []*metadatapb.CheckConstraintMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "test",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "table1",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "table1_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
								CheckConstraints: []*metadatapb.CheckConstraintMetadata{
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
			metadata: &metadatapb.DatabaseSchemaMetadata{
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{
							{
								Name: "users",
								Columns: []*metadatapb.ColumnMetadata{
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
								Indexes: []*metadatapb.IndexMetadata{
									{
										Name:        "users_pkey",
										Expressions: []string{"id"},
										Primary:     true,
									},
								},
								CheckConstraints: []*metadatapb.CheckConstraintMetadata{
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
			_, err = omnipg.Parse(result)
			require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
		})
	}
}

// TestCheckConstraintNotValidFormatNormalMode tests that CHECK constraints with NOT VALID
// are formatted correctly in normal (non-SDL) mode
func TestCheckConstraintNotValidFormatNormalMode(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "namespace_settings",
						Columns: []*metadatapb.ColumnMetadata{
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
						CheckConstraints: []*metadatapb.CheckConstraintMetadata{
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
	_, err = omnipg.Parse(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetDatabaseDefinitionSDLFormat_WithComments(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:    "test_schema",
				Comment: "Test schema for comments",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "users",
						Comment: "Users table with comments",
						Columns: []*metadatapb.ColumnMetadata{
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
						Indexes: []*metadatapb.IndexMetadata{
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
				Views: []*metadatapb.ViewMetadata{
					{
						Name:       "active_users",
						Definition: "SELECT id, name, email FROM test_schema.users WHERE active = true",
						Comment:    "View of active users",
					},
				},
				Functions: []*metadatapb.FunctionMetadata{
					{
						Name:       "get_user_count",
						Signature:  "get_user_count()",
						Definition: "CREATE FUNCTION test_schema.get_user_count() RETURNS INTEGER AS $$ BEGIN RETURN (SELECT COUNT(*) FROM test_schema.users); END; $$ LANGUAGE plpgsql",
						Comment:    "Function to get user count",
					},
				},
				Sequences: []*metadatapb.SequenceMetadata{
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
	assert.Contains(t, result, `COMMENT ON FUNCTION "test_schema".get_user_count() IS 'Function to get user count';`)

	// Verify sequence comment
	assert.Contains(t, result, `COMMENT ON SEQUENCE "test_schema"."custom_seq" IS 'Custom sequence for testing';`)

	// Verify index comment
	assert.Contains(t, result, `COMMENT ON INDEX "test_schema"."idx_users_email" IS 'Index on email column';`)

	// Validate that the generated SQL can be parsed without errors using ANTLR parser
	_, err = omnipg.Parse(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetDatabaseDefinitionSDLFormat_WithCommentsEscaping(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "test_table",
						Comment: "Table with 'single quotes' and \"double quotes\"",
						Columns: []*metadatapb.ColumnMetadata{
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
	_, err = omnipg.Parse(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetMultiFileDatabaseDefinition_WithComments(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name:    "app_schema",
				Comment: "Application schema",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "products",
						Comment: "Product catalog",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
								Comment:  "Product ID",
							},
							{
								Name:     "name",
								Type:     "TEXT",
								Nullable: false,
								Comment:  "Product name",
							},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{
								Name:        "idx_products_name",
								Expressions: []string{"name"},
								Unique:      false,
								Primary:     false,
								Comment:     "Index for product search",
							},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{
						Name:       "active_products",
						Definition: "SELECT id, name FROM app_schema.products WHERE active = true",
						Comment:    "View of active products",
					},
				},
				MaterializedViews: []*metadatapb.MaterializedViewMetadata{
					{
						Name:       "product_summary_mv",
						Definition: "SELECT id, name FROM app_schema.products",
						Comment:    "Materialized view of product summary",
					},
				},
				Functions: []*metadatapb.FunctionMetadata{
					{
						Name:       "count_products",
						Signature:  "count_products()",
						Definition: "CREATE FUNCTION app_schema.count_products() RETURNS INTEGER AS $$ BEGIN RETURN (SELECT COUNT(*) FROM app_schema.products); END; $$ LANGUAGE plpgsql",
						Comment:    "Returns product count",
					},
				},
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name:      "order_seq",
						DataType:  "bigint",
						Start:     "1",
						Increment: "1",
						MinValue:  "1",
						MaxValue:  "9223372036854775807",
						Cycle:     false,
						CacheSize: "1",
						Comment:   "Sequence for orders",
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetMultiFileDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Build a map for easier testing
	fileMap := make(map[string]string)
	for _, file := range result.Files {
		fileMap[file.Name] = file.Content
	}

	// Verify schema-level file with schema comment
	schemaFile, ok := fileMap["schemas/app_schema/schema.sql"]
	require.True(t, ok, "schema.sql file should exist for non-public schema")
	assert.Contains(t, schemaFile, `CREATE SCHEMA IF NOT EXISTS "app_schema";`)
	assert.Contains(t, schemaFile, `COMMENT ON SCHEMA "app_schema" IS 'Application schema';`)

	// Verify consolidated sequences file with comment (independent sequences go in sequences.sql)
	sequenceFile, ok := fileMap["schemas/app_schema/sequences.sql"]
	require.True(t, ok, "sequences.sql file should exist for independent sequences")
	assert.Contains(t, sequenceFile, `CREATE SEQUENCE "app_schema"."order_seq"`)
	assert.Contains(t, sequenceFile, `COMMENT ON SEQUENCE "app_schema"."order_seq" IS 'Sequence for orders';`)

	// Verify table file with comments
	tableFile, ok := fileMap["schemas/app_schema/tables/products.sql"]
	require.True(t, ok, "table file should exist")
	assert.Contains(t, tableFile, `CREATE TABLE "app_schema"."products"`)
	assert.Contains(t, tableFile, `COMMENT ON TABLE "app_schema"."products" IS 'Product catalog';`)
	assert.Contains(t, tableFile, `COMMENT ON COLUMN "app_schema"."products"."id" IS 'Product ID';`)
	assert.Contains(t, tableFile, `COMMENT ON COLUMN "app_schema"."products"."name" IS 'Product name';`)
	assert.Contains(t, tableFile, `COMMENT ON INDEX "app_schema"."idx_products_name" IS 'Index for product search';`)

	// Verify view file with comment
	viewFile, ok := fileMap["schemas/app_schema/views/active_products.sql"]
	require.True(t, ok, "view file should exist")
	assert.Contains(t, viewFile, `CREATE VIEW "app_schema"."active_products"`)
	assert.Contains(t, viewFile, `COMMENT ON VIEW "app_schema"."active_products" IS 'View of active products';`)

	// Verify materialized view file with comment
	materializedViewFile, ok := fileMap["schemas/app_schema/materialized_views/product_summary_mv.sql"]
	require.True(t, ok, "materialized view file should exist")
	assert.Contains(t, materializedViewFile, `CREATE MATERIALIZED VIEW "app_schema"."product_summary_mv"`)
	assert.Contains(t, materializedViewFile, `COMMENT ON MATERIALIZED VIEW "app_schema"."product_summary_mv" IS 'Materialized view of product summary';`)

	// Verify function file with comment
	functionFile, ok := fileMap["schemas/app_schema/functions/count_products.sql"]
	require.True(t, ok, "function file should exist")
	assert.Contains(t, functionFile, `CREATE FUNCTION app_schema.count_products()`)
	assert.Contains(t, functionFile, `COMMENT ON FUNCTION "app_schema".count_products() IS 'Returns product count';`)

	// Validate that each file's SQL can be parsed
	for fileName, content := range fileMap {
		_, err := omnipg.Parse(content)
		require.NoError(t, err, "SQL in file %s should be parseable by PostgreSQL parser", fileName)
	}
}

func TestGetMultiFileDatabaseDefinition_SchemaFileMakesCombinedSDLLoadable(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "metric_helpers",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "metrics",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "INTEGER",
								Nullable: false,
							},
						},
					},
				},
			},
		},
	}

	result, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, metadata)
	require.NoError(t, err)

	fileMap := make(map[string]string)
	var combined strings.Builder
	for _, file := range result.Files {
		fileMap[file.Name] = file.Content
		combined.WriteString(file.Content)
		combined.WriteString("\n")
	}

	schemaFile, ok := fileMap["schemas/metric_helpers/schema.sql"]
	assert.True(t, ok, "schema.sql file should exist for non-public schema")
	assert.Contains(t, schemaFile, `CREATE SCHEMA IF NOT EXISTS "metric_helpers";`)

	_, err = schema.DiffSDLMigration(storepb.Engine_POSTGRES, "", combined.String(), "")
	require.NoError(t, err)
}

func TestGetDatabaseDefinitionSDLFormat_SerialColumnWithSequence(t *testing.T) {
	// This test reproduces the issue where a SERIAL column causes duplicate CREATE SEQUENCE statements
	// SERIAL columns automatically create sequences, so we should NOT output separate CREATE SEQUENCE for them
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name:        "users_id_seq",
						DataType:    "integer",
						Start:       "1",
						Increment:   "1",
						MinValue:    "1",
						MaxValue:    "2147483647",
						Cycle:       false,
						OwnerTable:  "users",
						OwnerColumn: "id",
						Comment:     "Sequence for users id - should NOT appear in SDL",
					},
					{
						Name:       "independent_seq",
						DataType:   "bigint",
						Start:      "1",
						Increment:  "1",
						MinValue:   "1",
						MaxValue:   "9223372036854775807",
						Cycle:      false,
						OwnerTable: "", // Independent sequence - should appear in SDL
						Comment:    "Independent sequence - should appear in SDL",
					},
				},
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "integer",
								Nullable: false,
								Default:  "nextval('users_id_seq'::regclass)",
							},
							{
								Name:     "name",
								Type:     "text",
								Nullable: false,
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

	// The SDL should contain the table definition with serial type
	require.Contains(t, result, `CREATE TABLE "public"."users"`)
	require.Contains(t, result, `"id" serial`)
	require.Contains(t, result, `"name" text NOT NULL`)

	// The SDL SHOULD contain the independent sequence
	require.Contains(t, result, `CREATE SEQUENCE "public"."independent_seq"`,
		"SDL should contain CREATE SEQUENCE for independent sequences")

	// The SDL should NOT contain a separate CREATE SEQUENCE statement for users_id_seq
	// because the sequence is owned by the serial column (created implicitly by SERIAL type)
	require.NotContains(t, result, `CREATE SEQUENCE "public"."users_id_seq"`,
		"SDL should NOT contain CREATE SEQUENCE for sequence owned by a column (created implicitly by SERIAL)")
}

func TestGetMultiFileDatabaseDefinition_SerialColumnWithSequence(t *testing.T) {
	// This test verifies that multi-file format correctly handles serial columns and their sequences
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name:        "users_id_seq",
						DataType:    "integer",
						Start:       "1",
						Increment:   "1",
						MinValue:    "1",
						MaxValue:    "2147483647",
						Cycle:       false,
						OwnerTable:  "users",
						OwnerColumn: "id",
						Comment:     "Sequence for users id - should NOT appear as separate file",
					},
					{
						Name:       "independent_seq",
						DataType:   "bigint",
						Start:      "1",
						Increment:  "1",
						MinValue:   "1",
						MaxValue:   "9223372036854775807",
						Cycle:      false,
						OwnerTable: "", // Independent sequence - not owned by any column
						Comment:    "Independent sequence - should appear as separate file",
					},
					{
						Name:       "custom_seq",
						DataType:   "bigint",
						Start:      "100",
						Increment:  "5",
						MinValue:   "100",
						MaxValue:   "9223372036854775807",
						Cycle:      false,
						OwnerTable: "", // Not owned by a column (independent sequence)
						Comment:    "Custom sequence for orders - should appear as separate file",
					},
				},
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "integer",
								Nullable: false,
								Default:  "nextval('users_id_seq'::regclass)",
							},
							{
								Name:     "name",
								Type:     "text",
								Nullable: false,
							},
						},
					},
					{
						Name: "orders",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "integer",
								Nullable: false,
							},
							{
								Name:     "order_number",
								Type:     "bigint",
								Nullable: false,
								Default:  "nextval('custom_seq'::regclass)",
								Comment:  "Order number using custom sequence",
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

	result, err := GetMultiFileDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)

	// Build a map of files for easy lookup
	fileMap := make(map[string]string)
	for _, file := range result.Files {
		fileMap[file.Name] = file.Content
	}

	// Verify users table file exists and contains serial column
	usersFile, ok := fileMap["schemas/public/tables/users.sql"]
	require.True(t, ok, "users table file should exist")
	require.Contains(t, usersFile, `"id" serial`, "users table should use serial type")
	require.Contains(t, usersFile, `"name" text NOT NULL`)

	// Verify orders table file exists
	ordersFile, ok := fileMap["schemas/public/tables/orders.sql"]
	require.True(t, ok, "orders table file should exist")
	require.Contains(t, ordersFile, `"id" integer NOT NULL`)
	require.Contains(t, ordersFile, `"order_number" bigint DEFAULT nextval('custom_seq'::regclass) NOT NULL`,
		"order_number should use custom_seq via DEFAULT nextval()")

	// Verify the consolidated sequences file exists with both independent sequences
	sequencesFile, ok := fileMap["schemas/public/sequences.sql"]
	require.True(t, ok, "independent_seq file should exist")
	require.Contains(t, sequencesFile, `CREATE SEQUENCE "public"."independent_seq"`)
	require.Contains(t, sequencesFile, `COMMENT ON SEQUENCE "public"."independent_seq" IS 'Independent sequence - should appear as separate file'`)
	require.Contains(t, sequencesFile, `CREATE SEQUENCE "public"."custom_seq"`)
	require.Contains(t, sequencesFile, `COMMENT ON SEQUENCE "public"."custom_seq" IS 'Custom sequence for orders - should appear as separate file'`)
	require.NotContains(t, sequencesFile, `ALTER SEQUENCE`, "Independent sequences should not have ALTER SEQUENCE OWNED BY")

	// Verify individual sequence files do NOT exist (they should be in the consolidated file)
	_, ok = fileMap["schemas/public/sequences/independent_seq.sql"]
	require.False(t, ok, "independent_seq should NOT have individual file (should be in sequences.sql)")
	_, ok = fileMap["schemas/public/sequences/custom_seq.sql"]
	require.False(t, ok, "custom_seq should NOT have individual file (should be in sequences.sql)")

	// Verify users_id_seq does NOT appear in the sequences file (it's a serial sequence)
	require.NotContains(t, sequencesFile, "users_id_seq", "users_id_seq should NOT appear in sequences.sql because it's owned by a serial column")

	// Total files should be: 2 tables + 1 sequences file = 3 files
	require.Equal(t, 3, len(result.Files), "Should have exactly 3 files (2 tables + 1 consolidated sequences file)")
}

func TestGetDatabaseDefinitionSDLFormat_MultipleSequencesClaimingOwnership(t *testing.T) {
	// This test verifies that when multiple sequences claim ownership of the same column,
	// only the sequence referenced in the DEFAULT clause is skipped (treated as serial sequence).
	// The other sequences should still be output as CREATE SEQUENCE statements.
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Sequences: []*metadatapb.SequenceMetadata{
					{
						Name:        "test_sequence2",
						DataType:    "integer",
						Start:       "1",
						Increment:   "1",
						MinValue:    "1",
						MaxValue:    "2147483647",
						Cycle:       false,
						OwnerTable:  "test_table",
						OwnerColumn: "id",
					},
					{
						Name:        "test_table_id_seq",
						DataType:    "integer",
						Start:       "1",
						Increment:   "1",
						MinValue:    "1",
						MaxValue:    "2147483647",
						Cycle:       false,
						OwnerTable:  "test_table",
						OwnerColumn: "id",
					},
				},
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "test_table",
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "integer",
								Nullable: false,
								Default:  "nextval('test_table_id_seq'::regclass)",
							},
							{
								Name:     "name",
								Type:     "text",
								Nullable: false,
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

	expected := `CREATE SEQUENCE "public"."test_sequence2" AS integer START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 2147483647 NO CYCLE;

CREATE TABLE "public"."test_table" (
    "id" serial,
    "name" text NOT NULL
);

ALTER SEQUENCE "public"."test_sequence2" OWNED BY "public"."test_table"."id";

`

	assert.Equal(t, expected, result)

	// Verify that test_sequence2 is output (not skipped)
	assert.Contains(t, result, `CREATE SEQUENCE "public"."test_sequence2"`,
		"test_sequence2 should be output because it's not referenced in DEFAULT clause")

	// Verify that test_sequence2 has ALTER SEQUENCE OWNED BY statement
	assert.Contains(t, result, `ALTER SEQUENCE "public"."test_sequence2" OWNED BY "public"."test_table"."id"`,
		"test_sequence2 should have ALTER SEQUENCE OWNED BY because metadata indicates it's owned by the column")

	// Verify that test_table_id_seq is NOT output (skipped)
	assert.NotContains(t, result, `CREATE SEQUENCE "public"."test_table_id_seq"`,
		"test_table_id_seq should be skipped because it's referenced in DEFAULT clause and treated as serial")

	// Verify that the column uses serial type
	assert.Contains(t, result, `"id" serial`,
		"Column should use serial type because it references test_table_id_seq which is owned by it")

	// Validate that the generated SQL can be parsed
	_, err = omnipg.Parse(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetDatabaseDefinitionSDLFormat_ProcedureWithComment(t *testing.T) {
	// This test verifies that PROCEDURE comments are correctly generated with
	// COMMENT ON PROCEDURE (not COMMENT ON FUNCTION)
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*metadatapb.FunctionMetadata{
					{
						Name:      "get_user_count",
						Signature: "get_user_count()",
						Definition: `CREATE FUNCTION "public"."get_user_count"() RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT COUNT(*)::integer FROM users;
$$`,
						Comment: "Function to count users",
					},
					{
						Name:      "update_user_name",
						Signature: "update_user_name(user_id integer, new_name character varying)",
						Definition: `CREATE PROCEDURE "public"."update_user_name"(IN user_id integer, IN new_name character varying)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users
    SET name = new_name
    WHERE id = user_id;
END;
$$`,
						Comment: "Procedure to update user name",
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

	// Verify that FUNCTION has COMMENT ON FUNCTION
	assert.Contains(t, result, `COMMENT ON FUNCTION "public".get_user_count() IS 'Function to count users';`,
		"Function comment should use COMMENT ON FUNCTION")

	// Verify that PROCEDURE has COMMENT ON PROCEDURE (not COMMENT ON FUNCTION)
	assert.Contains(t, result, `COMMENT ON PROCEDURE "public".update_user_name(integer, character varying) IS 'Procedure to update user name';`,
		"Procedure comment should use COMMENT ON PROCEDURE with types-only signature")

	// Verify that we don't incorrectly use COMMENT ON FUNCTION for the procedure
	assert.NotContains(t, result, `COMMENT ON FUNCTION "public".update_user_name`,
		"Procedure comment should NOT use COMMENT ON FUNCTION")

	// Validate that the generated SQL can be parsed
	_, err = omnipg.Parse(result)
	require.NoError(t, err, "Generated SQL should be parseable by PostgreSQL parser")
}

func TestGetMultiFileDatabaseDefinition_FunctionAndProcedureSeparation(t *testing.T) {
	// This test verifies that functions and procedures are separated into different folders
	// in multi-file mode
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*metadatapb.FunctionMetadata{
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
						Definition: `CREATE FUNCTION "public"."get_user_by_id"(user_id integer) RETURNS TABLE(id integer, name character varying)
    LANGUAGE sql
    AS $$
    SELECT u.id, u.name FROM users u WHERE u.id = user_id;
$$`,
					},
					{
						Name: "update_user_name",
						Definition: `CREATE PROCEDURE "public"."update_user_name"(IN user_id integer, IN new_name character varying)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET name = new_name WHERE id = user_id;
END;
$$`,
					},
					{
						Name: "delete_old_users",
						Definition: `CREATE PROCEDURE "public"."delete_old_users"(IN days_old integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM users WHERE created_at < NOW() - INTERVAL '1 day' * days_old;
END;
$$`,
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SDLFormat: true,
	}

	result, err := GetMultiFileDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Build a map of files for easy lookup
	fileMap := make(map[string]string)
	for _, file := range result.Files {
		fileMap[file.Name] = file.Content
	}

	// Verify functions are in functions folder
	getUserCountFile, ok := fileMap["schemas/public/functions/get_user_count.sql"]
	require.True(t, ok, "get_user_count function should be in functions folder")
	assert.Contains(t, getUserCountFile, `CREATE FUNCTION "public"."get_user_count"()`)

	getUserByIDFile, ok := fileMap["schemas/public/functions/get_user_by_id.sql"]
	require.True(t, ok, "get_user_by_id function should be in functions folder")
	assert.Contains(t, getUserByIDFile, `CREATE FUNCTION "public"."get_user_by_id"`)

	// Verify procedures are in procedures folder (not functions folder)
	updateUserNameFile, ok := fileMap["schemas/public/procedures/update_user_name.sql"]
	require.True(t, ok, "update_user_name procedure should be in procedures folder")
	assert.Contains(t, updateUserNameFile, `CREATE PROCEDURE "public"."update_user_name"`)

	deleteOldUsersFile, ok := fileMap["schemas/public/procedures/delete_old_users.sql"]
	require.True(t, ok, "delete_old_users procedure should be in procedures folder")
	assert.Contains(t, deleteOldUsersFile, `CREATE PROCEDURE "public"."delete_old_users"`)

	// Verify procedures are NOT in functions folder
	_, ok = fileMap["schemas/public/functions/update_user_name.sql"]
	require.False(t, ok, "update_user_name should NOT be in functions folder")

	_, ok = fileMap["schemas/public/functions/delete_old_users.sql"]
	require.False(t, ok, "delete_old_users should NOT be in functions folder")

	// Total files should be: 2 functions + 2 procedures = 4 files
	require.Equal(t, 4, len(result.Files), "Should have exactly 4 files (2 functions + 2 procedures)")

	// Validate that each file's SQL can be parsed
	for fileName, content := range fileMap {
		_, err := omnipg.Parse(content)
		require.NoError(t, err, "SQL in file %s should be parseable by PostgreSQL parser", fileName)
	}
}

func TestOverloadedFunctionsPreservedInDump(t *testing.T) {
	oneArgDef := `CREATE OR REPLACE FUNCTION "public"."camel_to_snake"(camel_cased_text character varying)
 RETURNS character varying
 LANGUAGE plpgsql
 IMMUTABLE
AS $function$
BEGIN
  RETURN lower(camel_cased_text);
END;
$function$`
	twoArgDef := `CREATE OR REPLACE FUNCTION "public"."camel_to_snake"(camel_cased_text character varying, case_directive character varying)
 RETURNS character varying
 LANGUAGE plpgsql
 IMMUTABLE
AS $function$
BEGIN
  RETURN lower(camel_cased_text || '_' || case_directive);
END;
$function$`

	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*metadatapb.FunctionMetadata{
					{
						Name:       "camel_to_snake",
						Signature:  "camel_to_snake(camel_cased_text character varying)",
						Definition: oneArgDef,
					},
					{
						Name:       "camel_to_snake",
						Signature:  "camel_to_snake(camel_cased_text character varying, case_directive character varying)",
						Definition: twoArgDef,
					},
				},
			},
		},
	}

	oneArgSubstring := `"camel_to_snake"(camel_cased_text character varying)`
	twoArgSubstring := `"camel_to_snake"(camel_cased_text character varying, case_directive character varying)`

	// Test non-SDL format (GetDatabaseDefinition)
	result, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
	require.NoError(t, err)
	assert.Contains(t, result, oneArgSubstring)
	assert.Contains(t, result, twoArgSubstring)
	assert.Equal(t, 2, strings.Count(result, "CREATE OR REPLACE FUNCTION"), "both overloads must survive in non-SDL dump")

	// Test SDL format
	resultSDL, err := GetDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, metadata)
	require.NoError(t, err)
	assert.Contains(t, resultSDL, oneArgSubstring)
	assert.Contains(t, resultSDL, twoArgSubstring)
	assert.Equal(t, 2, strings.Count(resultSDL, "CREATE OR REPLACE FUNCTION"), "both overloads must survive in SDL dump")

	// Test single-schema path (GetSchemaDefinition)
	resultSchema, err := GetSchemaDefinition(metadata.Schemas[0])
	require.NoError(t, err)
	assert.Contains(t, resultSchema, oneArgSubstring)
	assert.Contains(t, resultSchema, twoArgSubstring)
	assert.Equal(t, 2, strings.Count(resultSchema, "CREATE OR REPLACE FUNCTION"), "both overloads must survive in single-schema dump")
}

// TestGetDatabaseDefinition_IndexCommentUsesRealNewline reproduces the A1
// bug where writeIndexComment used a Go raw string literal with `\n\n`,
// producing four literal characters `\`, `n`, `\`, `n` instead of two line
// feeds. The regression surfaced as omni pgparser choking on the backslash
// right after `COMMENT ON INDEX ... IS '...';`.
func TestGetDatabaseDefinition_IndexCommentUsesRealNewline(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "t",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "integer", Nullable: false},
				},
				Indexes: []*metadatapb.IndexMetadata{{
					Name:        "idx_t_id",
					Type:        "btree",
					Expressions: []string{"id"},
					Descending:  []bool{false},
					Definition:  "CREATE INDEX idx_t_id ON public.t USING btree (id);",
					Comment:     "sample comment",
				}},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetDatabaseDefinition: %v", err)
	}

	if strings.Contains(ddl, `\n`) {
		t.Errorf("DDL contains literal backslash-n escape; got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `COMMENT ON INDEX "public"."idx_t_id" IS 'sample comment';`) {
		t.Errorf("expected COMMENT ON INDEX statement, got:\n%s", ddl)
	}

	cat := catalog.New()
	if _, execErr := cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); execErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", execErr, ddl)
	}
}

// TestGetDatabaseDefinition_TableWithOnlyCheckConstraints reproduces the A2
// bug where a table with zero columns but with CHECK constraints produced
// `CREATE TABLE "..."."..." (,` — an invalid leading comma. We force the
// case by declaring a CHECK constraint on a table with no regular columns.
func TestGetDatabaseDefinition_TableWithOnlyCheckConstraints(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "cons_only",
				CheckConstraints: []*metadatapb.CheckConstraintMetadata{{
					Name:       "cons_only_dummy",
					Expression: "(1 = 1)",
				}},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetDatabaseDefinition: %v", err)
	}
	if strings.Contains(ddl, "(,") {
		t.Errorf("DDL still has `(,` leading comma; got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CREATE TABLE "public"."cons_only" (`) {
		t.Errorf("expected CREATE TABLE for cons_only, got:\n%s", ddl)
	}
	// A genuine zero-column table (no indexes or foreign keys) must keep its
	// column-independent CHECK constraint — only privilege-filtered broken
	// metadata goes down the bare-table path.
	if !strings.Contains(ddl, "cons_only_dummy") {
		t.Errorf("expected CHECK constraint cons_only_dummy to be preserved, got:\n%s", ddl)
	}
	// Omni will reject a table with zero columns semantically, but it must
	// at least parse cleanly; the test asserts the parse-level contract.
	if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", parseErr, ddl)
	}
}

// TestGetDatabaseDefinition_TableWithConstraintsButNoColumns reproduces a
// dump replay failure observed on a real workspace export: the sync user
// lacked column privileges on some tables, so information_schema.columns
// returned nothing while pg_catalog still exposed the tables' constraints.
// The dump then emitted `CREATE TABLE "s1"."t1" ();` followed by
// `ALTER TABLE ... ADD CONSTRAINT ... PRIMARY KEY (c1, c2);`, which
// PostgreSQL rejects because the columns do not exist. The dump must skip
// column-dependent DDL for such tables instead of producing non-replayable
// statements.
func TestGetDatabaseDefinition_TableWithConstraintsButNoColumns(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "s1",
			Sequences: []*metadatapb.SequenceMetadata{
				{
					// Owned by a column missing from t1's metadata: the sequence
					// itself must dump, its OWNED BY clause must not.
					Name:        "t1_c1_seq",
					DataType:    "bigint",
					Start:       "1",
					Increment:   "1",
					MinValue:    "1",
					MaxValue:    "9223372036854775807",
					OwnerTable:  "t1",
					OwnerColumn: "c1",
				},
				{
					// Owned by a column of t4, a zero-column table with no
					// broken-metadata signal (no indexes/FKs/partitions):
					// OWNED BY must still be skipped — a sequence cannot be
					// owned by a column of a zero-column table.
					Name:        "t4_id_seq",
					DataType:    "bigint",
					Start:       "1",
					Increment:   "1",
					MinValue:    "1",
					MaxValue:    "9223372036854775807",
					OwnerTable:  "t4",
					OwnerColumn: "id",
				},
			},
			Tables: []*metadatapb.TableMetadata{
				{
					// Broken sync: constraints and indexes, but no columns.
					Name: "t1",
					Indexes: []*metadatapb.IndexMetadata{
						{
							Name:         "pk_t1",
							Primary:      true,
							Unique:       true,
							IsConstraint: true,
							Expressions:  []string{"c1", "c2"},
						},
						{
							Name:         "uk_t1_c1",
							Unique:       true,
							IsConstraint: true,
							Expressions:  []string{"c1"},
						},
						{
							Name:        "idx_t1_c2",
							Type:        "btree",
							Expressions: []string{"c2"},
							Definition:  `CREATE INDEX idx_t1_c2 ON s1.t1 USING btree (c2);`,
							Comment:     "comment on a never-created index",
						},
					},
					CheckConstraints: []*metadatapb.CheckConstraintMetadata{{
						Name:       "chk_t1_c1",
						Expression: "(c1 > 0)",
					}},
					ForeignKeys: []*metadatapb.ForeignKeyMetadata{{
						Name:              "fk_t1_t2",
						Columns:           []string{"c1"},
						ReferencedSchema:  "s1",
						ReferencedTable:   "t2",
						ReferencedColumns: []string{"id"},
					}},
					Triggers: []*metadatapb.TriggerMetadata{{
						Name:    "trg_t1_c1",
						Body:    `CREATE TRIGGER trg_t1_c1 BEFORE UPDATE OF c1 ON s1.t1 FOR EACH ROW EXECUTE FUNCTION s1.f()`,
						Comment: "comment on a never-created trigger",
					}},
					Rules: []*metadatapb.RuleMetadata{{
						Name:       "rule_t1",
						Event:      "INSERT",
						Definition: `CREATE RULE rule_t1 AS ON INSERT TO s1.t1 WHERE new.c1 > 0 DO INSTEAD NOTHING;`,
					}},
				},
				{
					// Healthy table; its own constraints must still dump, but
					// its foreign key into the broken table must not.
					Name: "t2",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "integer", Nullable: false},
						{Name: "t1_c1", Type: "integer", Nullable: true},
					},
					Indexes: []*metadatapb.IndexMetadata{{
						Name:         "pk_t2",
						Primary:      true,
						Unique:       true,
						IsConstraint: true,
						Expressions:  []string{"id"},
					}},
					ForeignKeys: []*metadatapb.ForeignKeyMetadata{{
						Name:              "fk_t2_t1",
						Columns:           []string{"t1_c1"},
						ReferencedSchema:  "s1",
						ReferencedTable:   "t1",
						ReferencedColumns: []string{"c1"},
					}},
				},
				{
					// Broken sync on a partitioned table with no indexes: the
					// partition key references missing columns, so the table
					// must dump bare, without the PARTITION BY clause.
					Name: "t3",
					Partitions: []*metadatapb.TablePartitionMetadata{{
						Name:       "t3_p1",
						Expression: "RANGE (created_at)",
					}},
				},
				{
					// Genuine zero-column table: no indexes, foreign keys, or
					// partitions. Its column-independent CHECK must be kept.
					Name: "t4",
					CheckConstraints: []*metadatapb.CheckConstraintMetadata{{
						Name:       "chk_t4_ok",
						Expression: "(1 = 1)",
					}},
				},
			},
		}},
	}

	banned := []string{"pk_t1", "uk_t1_c1", "idx_t1_c2", "chk_t1_c1", "fk_t1_t2", "fk_t2_t1", "trg_t1_c1", "rule_t1", "t3_p1", "PARTITION BY", "OWNED BY", "COMMENT ON INDEX"}

	for name, ctx := range map[string]schema.GetDefinitionContext{
		"dump": {},
		"sdl":  {SDLFormat: true},
	} {
		ddl, err := GetDatabaseDefinition(ctx, meta)
		if err != nil {
			t.Fatalf("[%s] GetDatabaseDefinition: %v", name, err)
		}

		if !strings.Contains(ddl, `CREATE TABLE "s1"."t1" (`) {
			t.Errorf("[%s] expected bare CREATE TABLE for t1, got:\n%s", name, ddl)
		}
		for _, b := range banned {
			if strings.Contains(ddl, b) {
				t.Errorf("[%s] DDL references %q, which depends on columns missing from t1's metadata; got:\n%s", name, b, ddl)
			}
		}
		if !strings.Contains(ddl, "pk_t2") {
			t.Errorf("[%s] expected pk_t2 on the healthy table, got:\n%s", name, ddl)
		}
		if !strings.Contains(ddl, `CREATE SEQUENCE "s1"."t1_c1_seq"`) {
			t.Errorf("[%s] expected CREATE SEQUENCE for t1_c1_seq, got:\n%s", name, ddl)
		}
		if !strings.Contains(ddl, `CREATE TABLE "s1"."t3" (`) {
			t.Errorf("[%s] expected bare CREATE TABLE for t3, got:\n%s", name, ddl)
		}
		if !strings.Contains(ddl, "chk_t4_ok") {
			t.Errorf("[%s] expected genuine zero-column table t4 to keep its CHECK constraint, got:\n%s", name, ddl)
		}

		if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
			t.Errorf("[%s] omni catalog parse error: %v\nDDL:\n%s", name, parseErr, ddl)
		}
	}

	multiFile, err := GetMultiFileDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetMultiFileDatabaseDefinition: %v", err)
	}
	var combined strings.Builder
	for _, file := range multiFile.Files {
		combined.WriteString(file.Content)
	}
	for _, b := range banned {
		if strings.Contains(combined.String(), b) {
			t.Errorf("[multi-file] DDL references %q, which depends on columns missing from t1's metadata; got:\n%s", b, combined.String())
		}
	}
}

// TestGetDatabaseDefinition_TableNameContainingDot reproduces A3: a table
// whose name literally contains a period was writing out as `"".".."` —
// because the schema + "." + object object-ID was split back on the first
// dot and produced an empty schema. The fix uses a NUL separator.
func TestGetDatabaseDefinition_TableNameContainingDot(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "weird.name",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "c", Type: "text", Nullable: true},
				},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	if err != nil {
		t.Fatalf("GetDatabaseDefinition: %v", err)
	}
	if strings.Contains(ddl, `""."`) {
		t.Errorf("DDL contains empty schema `\"\".\"` prefix; got:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CREATE TABLE "public"."weird.name"`) {
		t.Errorf("expected schema to be public, got:\n%s", ddl)
	}
	if _, parseErr := catalog.New().Exec(ddl, &catalog.ExecOptions{ContinueOnError: true}); parseErr != nil {
		t.Errorf("omni catalog parse error: %v\nDDL:\n%s", parseErr, ddl)
	}
}

type getDatabaseDefinitionCase struct {
	Description string `yaml:"description"`
	Metadata    string `yaml:"metadata"`
	Expected    string `yaml:"expected"`
}

// TestGetDatabaseDefinition pins the DDL generated for each metadata input,
// and reloads that DDL into an omni catalog to prove it is valid PostgreSQL.
//
// The input is metadata rather than a schema text on purpose: deriving it by
// parsing would cap this test's reach at whatever the loader happens to extract,
// so any generator branch nothing feeds would go untested.
func TestGetDatabaseDefinition(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/get_database_definition.yaml"
	)

	var tests []getDatabaseDefinitionCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			var metadata metadatapb.DatabaseSchemaMetadata
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), &metadata))

			definition, err := GetDatabaseDefinition(schema.GetDefinitionContext{PrintHeader: true}, &metadata)
			require.NoError(t, err)
			require.NotEmpty(t, definition)

			_, err = catalog.New().Exec(definition, &catalog.ExecOptions{ContinueOnError: true})
			require.NoError(t, err, "generated definition should load into the omni catalog")

			// A YAML block scalar cannot carry a leading newline, so the blank
			// line this header opens with is trimmed rather than pinned.
			definition = strings.TrimPrefix(definition, "\n")

			if record {
				tests[i].Expected = definition
				return
			}
			require.Equal(t, tc.Expected, definition)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

func TestCheckConstraintOrderStability(t *testing.T) {
	// Simulate database metadata with multiple CHECK constraints
	// (similar to the packages_debian_group_distributions table shown in the screenshot)
	dbMetadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "packages_debian_group_distributions",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "bigint", Nullable: false},
							{Name: "suite", Type: "character varying(255)", Nullable: false},
							{Name: "codename", Type: "character varying(255)", Nullable: false},
							{Name: "signed_file", Type: "text", Nullable: true},
							{Name: "description", Type: "character varying(255)", Nullable: true},
							{Name: "origin", Type: "character varying(255)", Nullable: true},
							{Name: "file", Type: "character varying(255)", Nullable: true},
							{Name: "label", Type: "character varying(255)", Nullable: true},
							{Name: "version", Type: "character varying(255)", Nullable: true},
							{Name: "file_signature", Type: "character varying(4096)", Nullable: true},
						},
						CheckConstraints: []*metadatapb.CheckConstraintMetadata{
							{Name: "check_e7c928a24b", Expression: "(char_length((suite)::text) <= 255)"},
							{Name: "check_590e18405a", Expression: "(char_length((codename)::text) <= 255)"},
							{Name: "check_0007e0bf61", Expression: "(char_length(signed_file) <= 255)"},
							{Name: "check_310ac457b8", Expression: "(char_length((description)::text) <= 255)"},
							{Name: "check_3d6f87fc31", Expression: "(char_length((file_signature)::text) <= 4096)"},
							{Name: "check_3fdadf4a0c", Expression: "(char_length((version)::text) <= 255)"},
							{Name: "check_b057cd840a", Expression: "(char_length((origin)::text) <= 255)"},
							{Name: "check_be5ed8d307", Expression: "(char_length((file)::text) <= 255)"},
							{Name: "check_d3244bfc0b", Expression: "(char_length((label)::text) <= 255)"},
						},
					},
				},
			},
		},
	}

	ctx := schema.GetDefinitionContext{
		SkipBackupSchema: false,
		PrintHeader:      false,
		SDLFormat:        true,
	}

	// Generate SDL first time
	sdl1, err := GetDatabaseDefinition(ctx, dbMetadata)
	require.NoError(t, err)
	require.NotEmpty(t, sdl1)

	t.Logf("First SDL generation:\n%s", sdl1)

	// Generate SDL second time from the same metadata
	sdl2, err := GetDatabaseDefinition(ctx, dbMetadata)
	require.NoError(t, err)
	require.NotEmpty(t, sdl2)

	t.Logf("Second SDL generation:\n%s", sdl2)

	// The two SDL outputs should be identical
	require.Equal(t, sdl1, sdl2, "SDL generation should be deterministic - CHECK constraint order should be stable")
}

func TestGetDatabaseDefinitionSDLFormat_CompositeTypes(t *testing.T) {
	// aa_nested sorts before its dependency zz_base alphabetically, so correct
	// output proves dependency ordering rather than name ordering.
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*metadatapb.EnumTypeMetadata{
					{Name: "order_status", Values: []string{"pending", "shipped"}},
				},
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						Name: "aa_nested",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "home", Type: "public.zz_base"},
							{Name: "homes", Type: "public.zz_base[]"},
							{Name: "status", Type: "public.order_status"},
						},
					},
					{
						Name:    "zz_base",
						Comment: "base address type",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "street", Type: "text", Collation: `"C"`, Comment: "street line"},
							{Name: "city", Type: "character varying(50)"},
						},
					},
					{
						Name:     "ext_owned",
						SkipDump: true,
						Attributes: []*metadatapb.CompositeTypeAttribute{
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
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						Name: "point2",
						Attributes: []*metadatapb.CompositeTypeAttribute{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*metadatapb.EnumTypeMetadata{
					{Name: "mood", Values: []string{"happy", "sad"}},
				},
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						Name: "wrapper",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "b", Type: "public.base"},
						},
					},
					{
						Name: "base",
						Attributes: []*metadatapb.CompositeTypeAttribute{
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
	c1 := &metadatapb.CompositeTypeMetadata{
		Name:       "c1",
		Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "x", Type: "integer"}},
	}
	b2 := &metadatapb.CompositeTypeMetadata{
		Name:       "b2",
		Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "c", Type: "public.c1"}},
	}
	a3 := &metadatapb.CompositeTypeMetadata{
		Name:       "a3",
		Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "b", Type: "other.b2[]"}},
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
		{Schema: "public", Composite: &metadatapb.CompositeTypeMetadata{Name: "zz"}},
		{Schema: "public", Composite: &metadatapb.CompositeTypeMetadata{Name: "aa"}},
	})
	require.Len(t, ordered, 2)
	assert.Equal(t, "aa", ordered[0].Composite.Name)
	assert.Equal(t, "zz", ordered[1].Composite.Name)
}

func TestEventTriggerSDLSingleFileOutput(t *testing.T) {
	metadata := eventTriggerSDLMetadata()

	result, err := GetDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, metadata)
	require.NoError(t, err)
	assert.Contains(t, result, `CREATE FUNCTION "public"."audit_ddl"() RETURNS event_trigger`)
	assert.Contains(t, result, `CREATE EVENT TRIGGER "audit_ddl_start" ON ddl_command_start`)
	assert.Contains(t, result, `WHEN TAG IN ('CREATE TABLE')`)
	assert.Contains(t, result, `EXECUTE FUNCTION "public"."audit_ddl"();`)
	assert.Contains(t, result, `COMMENT ON EVENT TRIGGER "audit_ddl_start" IS 'Audit DDL start';`)

	sdl, err := schema.MetadataToSDL(
		storepb.Engine_POSTGRES,
		model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true),
	)
	require.NoError(t, err)
	assert.Contains(t, sdl, `CREATE EVENT TRIGGER "audit_ddl_start" ON ddl_command_start`)
	assert.Contains(t, sdl, `COMMENT ON EVENT TRIGGER "audit_ddl_start" IS 'Audit DDL start';`)
}

func TestEventTriggerSDLMultiFileOutput(t *testing.T) {
	result, err := GetMultiFileDatabaseDefinition(
		schema.GetDefinitionContext{SDLFormat: true},
		eventTriggerSDLMetadata(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	fileMap := make(map[string]string)
	for _, file := range result.Files {
		fileMap[file.Name] = file.Content
	}

	eventTriggerFile, ok := fileMap["event_triggers.sql"]
	require.True(t, ok, "event_triggers.sql file should exist")
	assert.Contains(t, eventTriggerFile, `CREATE EVENT TRIGGER "audit_ddl_start" ON ddl_command_start`)
	assert.Contains(t, eventTriggerFile, `WHEN TAG IN ('CREATE TABLE')`)
	assert.Contains(t, eventTriggerFile, `EXECUTE FUNCTION "public"."audit_ddl"();`)
	assert.Contains(t, eventTriggerFile, `COMMENT ON EVENT TRIGGER "audit_ddl_start" IS 'Audit DDL start';`)
}

func eventTriggerSDLMetadata() *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*metadatapb.FunctionMetadata{
					{
						Name:      "audit_ddl",
						Signature: "audit_ddl()",
						Definition: `CREATE FUNCTION "public"."audit_ddl"() RETURNS event_trigger
LANGUAGE plpgsql
AS $$
BEGIN
END;
$$`,
					},
				},
			},
		},
		EventTriggers: []*metadatapb.EventTriggerMetadata{
			{
				Name:           "audit_ddl_start",
				Event:          "ddl_command_start",
				Tags:           []string{"CREATE TABLE"},
				FunctionSchema: "public",
				FunctionName:   "audit_ddl",
				Enabled:        true,
				Comment:        "Audit DDL start",
			},
		},
	}
}

func TestTriggerSDLSingleFileOutput(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*metadatapb.TriggerMetadata{
							{
								Name: "audit_trigger",
								Body: "CREATE TRIGGER audit_trigger AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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

	assert.Contains(t, result, "CREATE TRIGGER audit_trigger")
	assert.Contains(t, result, "AFTER INSERT ON public.users")
	assert.Contains(t, result, "FOR EACH ROW EXECUTE FUNCTION audit_log()")
}

func TestTriggerSDLMultiFileOutput(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*metadatapb.TriggerMetadata{
							{
								Name: "audit_trigger",
								Body: "CREATE TRIGGER audit_trigger AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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

	result, err := GetMultiFileDatabaseDefinition(ctx, metadata)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the users table file
	var usersFileContent string
	for _, file := range result.Files {
		if file.Name == "schemas/public/tables/users.sql" {
			usersFileContent = file.Content
			break
		}
	}

	require.NotEmpty(t, usersFileContent, "Should have users table file")
	assert.Contains(t, usersFileContent, "CREATE TRIGGER audit_trigger")
	assert.Contains(t, usersFileContent, "AFTER INSERT ON public.users")
}

func TestTriggerSDLWithCompleteDefinition(t *testing.T) {
	// Test that triggers with complete CREATE TRIGGER statements in Body are output correctly
	// This simulates triggers dumped from the database where Body contains the full statement
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "ci_builds",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*metadatapb.TriggerMetadata{
							{
								Name: "ci_builds_loose_fk_trigger",
								Body: "CREATE TRIGGER ci_builds_loose_fk_trigger AFTER DELETE ON public.ci_builds REFERENCING OLD TABLE AS old_table FOR EACH STATEMENT EXECUTE FUNCTION public.loose_foreign_key_on_builds_projects()",
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

	// Should contain only one CREATE TRIGGER (not duplicated)
	createTriggerCount := strings.Count(result, "CREATE TRIGGER ci_builds_loose_fk_trigger")
	assert.Equal(t, 1, createTriggerCount, "CREATE TRIGGER should appear exactly once, not duplicated")

	// Should contain the complete trigger definition
	assert.Contains(t, result, "AFTER DELETE ON public.ci_builds")
	assert.Contains(t, result, "REFERENCING OLD TABLE AS old_table")
	assert.Contains(t, result, "FOR EACH STATEMENT")
	assert.Contains(t, result, "EXECUTE FUNCTION public.loose_foreign_key_on_builds_projects()")
}

func TestTriggerSDLWithComment(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*metadatapb.TriggerMetadata{
							{
								Name:    "audit_trigger",
								Body:    "CREATE TRIGGER audit_trigger AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
								Comment: "Audit log trigger",
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

	assert.Contains(t, result, "CREATE TRIGGER audit_trigger")
	assert.Contains(t, result, "COMMENT ON TRIGGER")
	assert.Contains(t, result, "audit_trigger")
	assert.Contains(t, result, "Audit log trigger")
}

func TestMultipleTriggersSDL(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
						Triggers: []*metadatapb.TriggerMetadata{
							{
								Name: "audit_insert",
								Body: "CREATE TRIGGER audit_insert AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
							},
							{
								Name: "audit_update",
								Body: "CREATE TRIGGER audit_update AFTER UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION audit_log()",
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

	assert.Contains(t, result, "audit_insert")
	assert.Contains(t, result, "audit_update")
}

func TestIsDefinitionProcedure(t *testing.T) {
	tests := []struct {
		name        string
		definition  string
		isProcedure bool
	}{
		{
			name: "Simple FUNCTION",
			definition: `CREATE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "Simple PROCEDURE",
			definition: `CREATE PROCEDURE test_procedure()
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: true,
		},
		{
			name: "CREATE OR REPLACE FUNCTION",
			definition: `CREATE OR REPLACE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "CREATE OR REPLACE PROCEDURE",
			definition: `CREATE OR REPLACE PROCEDURE test_procedure()
LANGUAGE plpgsql
AS $$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: true,
		},
		{
			name: "FUNCTION with word 'procedure' in comment - should NOT be confused",
			definition: `CREATE FUNCTION test_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	-- This is not a procedure, it's a function
	-- The word PROCEDURE appears in this comment
	RAISE NOTICE 'This function is not a procedure';
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "FUNCTION with word 'procedure' in string literal - should NOT be confused",
			definition: `CREATE FUNCTION test_function()
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
	RETURN 'This function is not a PROCEDURE';
END;
$$;`,
			isProcedure: false,
		},
		{
			name: "PROCEDURE with complex body",
			definition: `CREATE OR REPLACE PROCEDURE update_user_data(
	user_id INTEGER,
	new_name VARCHAR
)
LANGUAGE plpgsql
AS $$
DECLARE
	old_name VARCHAR;
BEGIN
	-- Get old name
	SELECT name INTO old_name FROM users WHERE id = user_id;

	-- Update the user name
	UPDATE users SET name = new_name WHERE id = user_id;

	-- Log the change
	INSERT INTO audit_log (message) VALUES (
		'Changed user name from ' || old_name || ' to ' || new_name
	);
END;
$$;`,
			isProcedure: true,
		},
		{
			name: "FUNCTION with RETURNS TABLE",
			definition: `CREATE OR REPLACE FUNCTION get_users()
RETURNS TABLE(id INTEGER, name VARCHAR)
LANGUAGE plpgsql
AS $$
BEGIN
	RETURN QUERY SELECT users.id, users.name FROM users;
END;
$$;`,
			isProcedure: false,
		},
		{
			name:        "Empty definition",
			definition:  "",
			isProcedure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDefinitionProcedure(tt.definition)
			assert.Equal(t, tt.isProcedure, result,
				"isDefinitionProcedure returned %v, expected %v for:\n%s",
				result, tt.isProcedure, tt.definition)
		})
	}
}

// TestIsDefinitionProcedureRobustness tests edge cases that would fail with string-based detection
func TestIsDefinitionProcedureRobustness(t *testing.T) {
	tests := []struct {
		name        string
		definition  string
		isProcedure bool
		reason      string
	}{
		{
			name: "FUNCTION with 'CREATE PROCEDURE' in string literal",
			definition: `CREATE FUNCTION test_function()
RETURNS text
LANGUAGE sql
AS $outer$
	SELECT 'Example: CREATE PROCEDURE foo() LANGUAGE plpgsql AS $body$ BEGIN NULL; END; $body$;'
$outer$;`,
			isProcedure: false,
			reason:      "String-based detection might incorrectly identify this as a PROCEDURE due to 'CREATE PROCEDURE' in the string literal",
		},
		{
			name: "FUNCTION with 'PROCEDURE' in multi-line comment",
			definition: `CREATE FUNCTION complex_function()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	/*
	 * This function performs complex operations
	 * Note: This is NOT a PROCEDURE
	 * PROCEDURE keyword appears in this comment
	 * But it should still be detected as a FUNCTION
	 */
	RAISE NOTICE 'Processing...';
END;
$$;`,
			isProcedure: false,
			reason:      "Multi-line comments with PROCEDURE keyword should not confuse the detector",
		},
		{
			name: "PROCEDURE with unusual formatting",
			definition: `CREATE    OR    REPLACE    PROCEDURE
test_procedure
(
)
LANGUAGE plpgsql
AS
$$
BEGIN
	NULL;
END;
$$;`,
			isProcedure: true,
			reason:      "Should handle unusual whitespace and formatting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDefinitionProcedure(tt.definition)
			assert.Equal(t, tt.isProcedure, result,
				"Test failed: %s\nGot: %v, Expected: %v",
				tt.reason, result, tt.isProcedure)
		})
	}
}

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
