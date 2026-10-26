import type {
  PlanTableFacts,
  PlanTableIndex,
  PlanTableLookup,
} from "@/apps/explain-visualizer/plan-model";
import type {
  DatabaseMetadata,
  IndexMetadata,
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

/**
 * What Bytebase's own metadata says about a table a plan node reads.
 *
 * A plan names a relation without its schema, so a name two schemas both use
 * is left unanswered rather than answered with whichever came first: the wrong
 * table's indexes are worse than none. The metadata is the last sync's, which
 * is what the rest of the editor shows too.
 */
export function tableFactsLookup(metadata: DatabaseMetadata): PlanTableLookup {
  return (relation) => {
    if (!relation) return undefined;
    const matches = metadata.schemas.flatMap((schema) =>
      schema.tables.filter((table) => table.name === relation)
    );
    return matches.length === 1 ? toFacts(matches[0]) : undefined;
  };
}
