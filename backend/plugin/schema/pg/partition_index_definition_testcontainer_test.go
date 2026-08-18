package pg

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

const partitionIndexRoundTripDDL = `
CREATE TABLE accounts (
    tenant text NOT NULL,
    id integer NOT NULL,
    email text,
    PRIMARY KEY (tenant, id)
) PARTITION BY LIST (tenant);
CREATE TABLE accounts_t1 PARTITION OF accounts FOR VALUES IN ('t1');
CREATE TABLE accounts_t2 PARTITION OF accounts FOR VALUES IN ('t2') PARTITION BY RANGE (id);
CREATE TABLE accounts_t2_a PARTITION OF accounts_t2 FOR VALUES FROM (0) TO (100);
CREATE INDEX accounts_email_idx ON accounts (email);
CREATE INDEX accounts_t1_local_idx ON accounts_t1 (email, id);
CREATE INDEX accounts_t2_local_idx ON accounts_t2 (id, email);
`

// TestGetDatabaseDefinitionPartitionIndexRoundTrip verifies that the generated
// definition of a partitioned table with indexes restores to a database where
// every index is valid and every partition carries its indexes.
func TestGetDatabaseDefinitionPartitionIndexRoundTrip(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	pgDB := pgContainer.GetDB()
	require.NoError(t, pgDB.Ping())

	dbNameA := fmt.Sprintf("part_idx_a_%d", time.Now().UnixNano()%1000000)
	_, err := pgDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbNameA))
	require.NoError(t, err)

	driverA, err := createPgDriver(ctx, pgContainer.GetHost(), pgContainer.GetPort(), dbNameA)
	require.NoError(t, err)
	defer driverA.Close(ctx)

	_, err = driverA.GetDB().ExecContext(ctx, partitionIndexRoundTripDDL)
	require.NoError(t, err)

	metadataA, err := driverA.SyncDBSchema(ctx)
	require.NoError(t, err)

	generatedDDL, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadataA)
	require.NoError(t, err)

	dbNameB := fmt.Sprintf("part_idx_b_%d", time.Now().UnixNano()%1000000)
	_, err = pgDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbNameB))
	require.NoError(t, err)

	driverB, err := createPgDriver(ctx, pgContainer.GetHost(), pgContainer.GetPort(), dbNameB)
	require.NoError(t, err)
	defer driverB.Close(ctx)

	_, err = driverB.GetDB().ExecContext(ctx, generatedDDL)
	require.NoError(t, err, "failed to execute generated DDL: %s", generatedDDL)

	dbB := driverB.GetDB()

	var invalidCount int
	require.NoError(t, dbB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pg_index WHERE NOT indisvalid`).Scan(&invalidCount))
	require.Zero(t, invalidCount, "restored database has invalid indexes; generated DDL:\n%s", generatedDDL)

	var indexInheritanceEdges int
	require.NoError(t, dbB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid WHERE c.relkind IN ('i', 'I')`).Scan(&indexInheritanceEdges))
	require.Equal(t, 7, indexInheritanceEdges, "generated DDL:\n%s", generatedDDL)

	for _, indexName := range []string{"accounts_pkey", "accounts_email_idx", "accounts_t1_local_idx", "accounts_t2_local_idx"} {
		var exists bool
		require.NoError(t, dbB.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_class WHERE relname = $1 AND relkind IN ('i', 'I'))`, indexName).Scan(&exists))
		require.True(t, exists, "index %s missing from restored database; generated DDL:\n%s", indexName, generatedDDL)
	}
}

// TestGetDatabaseDefinitionLegacyPartitionIndexMetadata pins the pg_dump-style
// output for metadata rows that still record inherited partition child indexes.
func TestGetDatabaseDefinitionLegacyPartitionIndexMetadata(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "accounts",
				Columns: []*storepb.ColumnMetadata{
					{Name: "tenant", Type: "text", Nullable: false},
					{Name: "email", Type: "text", Nullable: true},
				},
				Indexes: []*storepb.IndexMetadata{{
					Name:        "accounts_email_idx",
					Type:        "btree",
					Expressions: []string{"email"},
					Definition:  "CREATE INDEX accounts_email_idx ON ONLY public.accounts USING btree (email);",
				}},
				Partitions: []*storepb.TablePartitionMetadata{{
					Name:       "accounts_t1",
					Type:       storepb.TablePartitionMetadata_LIST,
					Expression: "LIST (tenant)",
					Value:      "FOR VALUES IN ('t1')",
					Indexes: []*storepb.IndexMetadata{{
						Name:              "accounts_t1_email_idx",
						Type:              "btree",
						Expressions:       []string{"email"},
						Definition:        "CREATE INDEX accounts_t1_email_idx ON public.accounts_t1 USING btree (email);",
						ParentIndexSchema: "public",
						ParentIndexName:   "accounts_email_idx",
					}},
				}},
			}},
		}},
	}

	ddl, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, meta)
	require.NoError(t, err)

	require.Contains(t, ddl, `CREATE INDEX "accounts_email_idx" ON ONLY "public"."accounts"`)
	require.Contains(t, ddl, `CREATE INDEX "accounts_t1_email_idx" ON "public"."accounts_t1"`)
	require.Contains(t, ddl, `ALTER INDEX "public"."accounts_email_idx" ATTACH PARTITION "public"."accounts_t1_email_idx";`)
	require.False(t, strings.Contains(ddl, `ON ONLY "public"."accounts_t1"`), "partition without subpartitions must not use ON ONLY; got:\n%s", ddl)
}
