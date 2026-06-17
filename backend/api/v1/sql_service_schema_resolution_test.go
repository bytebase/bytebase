package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestSchemaForWriteTargetResolution pins the per-engine default schema used to resolve
// UNqualified DML/DDL write targets — the schema the engine will actually write to, or the
// sentinel when that can't be determined ahead of execution (so resource.schema_name is
// omitted and schema-scoped grants fail closed). See SUP-222 / BYT-9698.
func TestSchemaForWriteTargetResolution(t *testing.T) {
	const dbName = "ORADB"
	tests := []struct {
		name          string
		engine        storepb.Engine
		requestSchema string
		want          string
	}{
		{"postgres with request schema resolves to it", storepb.Engine_POSTGRES, "other_schema", "other_schema"},
		{"postgres without request schema is unknowable ($user/public) → sentinel", storepb.Engine_POSTGRES, "", unresolvedSchemaSentinel},
		{"mssql is always sentinel (login default_schema; never trust a request schema)", storepb.Engine_MSSQL, "ignored", unresolvedSchemaSentinel},
		{"oracle pins CURRENT_SCHEMA to the database name", storepb.Engine_ORACLE, "", dbName},
		{"mysql has no schema layer → omit", storepb.Engine_MYSQL, "", ""},
		{"tidb has no schema layer → omit", storepb.Engine_TIDB, "", ""},
		// Any engine without an explicit, execution-verified mapping must fail closed via the
		// sentinel — never "" (which a pg-family extractor would fill from the metadata search
		// path, a real schema name we'd then wrongly assert). Guards a future newACL flip for
		// COCKROACHDB/REDSHIFT (both pg-family extractors). SUP-222 / BYT-9698.
		{"unlisted engine fails closed via sentinel", storepb.Engine_SNOWFLAKE, "ignored", unresolvedSchemaSentinel},
		{"cockroachdb (pg-family extractor) fails closed via sentinel", storepb.Engine_COCKROACHDB, "", unresolvedSchemaSentinel},
		{"redshift (pg-family extractor) fails closed via sentinel", storepb.Engine_REDSHIFT, "", unresolvedSchemaSentinel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, schemaForWriteTargetResolution(tc.engine, dbName, tc.requestSchema))
		})
	}
}

// TestUnresolvedSchemaSentinelIsImpossible guards that the sentinel can never collide with a
// real schema: a NUL byte is not representable in a PostgreSQL/MSSQL identifier.
func TestUnresolvedSchemaSentinelIsImpossible(t *testing.T) {
	require.Contains(t, unresolvedSchemaSentinel, "\x00")
}
