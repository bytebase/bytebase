package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
										Expression: "age >= 0",
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
    CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "users_email_key" UNIQUE ("email"),
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
    CONSTRAINT "orders_pkey" PRIMARY KEY ("id"),
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
    CONSTRAINT "products_pkey" PRIMARY KEY ("id")
);

CREATE INDEX "idx_products_name" ON "public"."products" ("name");

CREATE INDEX "idx_products_category_price" ON "public"."products" ("category_id", "price" DESC);

CREATE UNIQUE INDEX "idx_products_name_unique" ON "public"."products" ("name");

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
    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "public"."orders" (
    "id" SERIAL NOT NULL,
    "user_id" INTEGER NOT NULL,
    "total" DECIMAL(10,2) NOT NULL,
    CONSTRAINT "orders_pkey" PRIMARY KEY ("id")
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
    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
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
			expected: `CREATE SEQUENCE "public"."independent_seq";

CREATE SEQUENCE "public"."order_seq" AS integer START WITH 1000 INCREMENT BY 10 MINVALUE 1000 MAXVALUE 999999 CYCLE;

CREATE SEQUENCE "public"."user_id_seq";

CREATE TABLE "public"."users" (
    "id" INTEGER DEFAULT nextval('user_id_seq'::regclass) NOT NULL,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
