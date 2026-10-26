import type {
  PlanTableFacts,
  PlanTableIndex,
  PlanTableLookup,
} from "@/apps/explain-visualizer/plan-model";
import type {
  DatabaseMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

const toIndex = (index: IndexMetadata): PlanTableIndex => ({
  name: index.name,
  columns: index.expressions.join(", "),
  unique: index.unique,
  primary: index.primary,
});

const toFacts = (table: TableMetadata): PlanTableFacts => ({
  // A table nothing has counted reports zero, which is not a count of zero.
  rowCount: table.rowCount > 0n ? Number(table.rowCount) : undefined,
  indexes: table.indexes.map(toIndex),
});

/** Engines differ on identifier case, and a qualifier is not worth a mismatch. */
const sameIdentifier = (a: string, b: string) =>
  a.localeCompare(b, undefined, { sensitivity: "accent" }) === 0;

/**
 * What Bytebase's own metadata says about a table a plan node reads.
 *
 * The metadata is the connected database's alone, so a plan that names another
 * database — SQL Server writes the database on every object it reads, and a
 * query can cross databases — is left unanswered. Answering it from here would
 * describe a different table that happens to share a name, and the wrong
 * table's indexes are worse than none.
 *
 * The same holds within a database: a plan that names no schema and a name two
 * schemas both use is left unanswered. The metadata is the last sync's, which
 * is what the rest of the editor shows too.
 */
export function tableFactsLookup(
  metadata: DatabaseMetadata,
  connectedDatabase: string
): PlanTableLookup {
  return ({ table, schema, database }) => {
    if (!table) return undefined;
    if (database && !sameIdentifier(database, connectedDatabase)) {
      return undefined;
    }
    const schemas: SchemaMetadata[] = schema
      ? metadata.schemas.filter((entry) => sameIdentifier(entry.name, schema))
      : metadata.schemas;
    const matches = schemas.flatMap((entry) =>
      entry.tables.filter((entry) => entry.name === table)
    );
    return matches.length === 1 ? toFacts(matches[0]) : undefined;
  };
}
