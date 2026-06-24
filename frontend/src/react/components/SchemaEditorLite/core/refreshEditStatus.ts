import { isEqual, pick } from "lodash-es";
import type {
  Database,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  ComparableColumnFields,
  ComparableForeignKeyFields,
  ComparableIndexFields,
  ComparableTableFields,
  ComparableTablePartitionFields,
} from "@/utils";
import type { EditStatusContext } from "../types";
import { keyForResource } from "./keyForResource";

function isEqualForeignKeys(
  source: ForeignKeyMetadata[],
  target: ForeignKeyMetadata[]
): boolean {
  if (source.length !== target.length) return false;
  return source.every((fk, i) =>
    isEqual(
      pick(fk, ComparableForeignKeyFields),
      pick(target[i], ComparableForeignKeyFields)
    )
  );
}

function isEqualIndexes(
  source: IndexMetadata[],
  target: IndexMetadata[]
): boolean {
  if (source.length !== target.length) return false;
  const targetByName = new Map(target.map((idx) => [idx.name, idx]));
  return source.every((sourceIndex) => {
    const targetIndex = targetByName.get(sourceIndex.name);
    return (
      !!targetIndex &&
      isEqual(
        pick(sourceIndex, ComparableIndexFields),
        pick(targetIndex, ComparableIndexFields)
      )
    );
  });
}

function isEqualPartitions(
  source: TablePartitionMetadata[],
  target: TablePartitionMetadata[]
): boolean {
  if (source.length !== target.length) return false;
  const targetByName = new Map(target.map((part) => [part.name, part]));
  return source.every((sourcePartition) => {
    const targetPartition = targetByName.get(sourcePartition.name);
    return (
      !!targetPartition &&
      isEqual(
        pick(sourcePartition, ComparableTablePartitionFields),
        pick(targetPartition, ComparableTablePartitionFields)
      ) &&
      isEqualPartitions(
        sourcePartition.subpartitions ?? [],
        targetPartition.subpartitions ?? []
      )
    );
  });
}

/**
 * Recompute the edit status of a single table and its columns by diffing the
 * current metadata against the baseline, so reverting an edit back to its
 * original value clears the dirty marker. Unlike `DiffMerge`/`rebuildMetadataEdit`
 * this is non-mutating (it never re-inserts dropped rows or reorders), so it's
 * safe to call on every keystroke. It's scoped to the one edited table to stay
 * cheap.
 *
 * `created`/`dropped` resources are left untouched — those are structural
 * changes, not field edits, and must not be cleared by a value comparison.
 */
export function refreshTableEditStatus(
  editStatus: EditStatusContext,
  db: Database,
  baselineMetadata: DatabaseMetadata,
  schema: SchemaMetadata,
  table: TableMetadata
): void {
  const ownStatus = editStatus.getEditStatusByKey(
    keyForResource(db, { schema, table })
  );
  if (ownStatus !== "created" && ownStatus !== "dropped") {
    const baselineTable = baselineMetadata.schemas
      .find((s) => s.name === schema.name)
      ?.tables.find((t) => t.name === table.name);

    // The table's own status reflects its scalar fields plus indexes, foreign
    // keys, and partitions. Index/FK/partition edits are all tracked on the
    // table key (see TableEditor's add handlers), so partitions must be
    // compared here too — otherwise a reverted column/comment edit could clear
    // the only marker for an added partition. Column edits surface separately
    // via the per-column statuses below.
    const ownChanged =
      !baselineTable ||
      !isEqual(
        pick(table, ComparableTableFields),
        pick(baselineTable, ComparableTableFields)
      ) ||
      !isEqualForeignKeys(baselineTable.foreignKeys, table.foreignKeys) ||
      !isEqualIndexes(baselineTable.indexes, table.indexes) ||
      !isEqualPartitions(baselineTable.partitions, table.partitions);
    if (ownChanged) {
      editStatus.markEditStatus(db, { schema, table }, "updated");
    } else {
      // Non-recursive: clear only the table's own key so the per-column
      // statuses (set below) survive.
      editStatus.removeEditStatus(db, { schema, table }, false);
    }

    // Edit-status keys are column-name based, so two columns sharing a name
    // (an in-progress rename collision) map to the same key. Treat any such
    // duplicate as changed and never clear its key — otherwise the unchanged
    // twin could remove the dirty marker the renamed twin just set, leaving
    // the editor wrongly clean while the metadata actually differs.
    const nameCounts = new Map<string, number>();
    for (const column of table.columns) {
      nameCounts.set(column.name, (nameCounts.get(column.name) ?? 0) + 1);
    }

    for (const column of table.columns) {
      const columnStatus = editStatus.getColumnStatus(db, {
        schema,
        table,
        column,
      });
      if (columnStatus === "created" || columnStatus === "dropped") continue;
      const baselineColumn = baselineTable?.columns.find(
        (c) => c.name === column.name
      );
      const changed =
        (nameCounts.get(column.name) ?? 0) > 1 ||
        !baselineColumn ||
        !isEqual(
          pick(column, ComparableColumnFields),
          pick(baselineColumn, ComparableColumnFields)
        );
      if (changed) {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      } else {
        editStatus.removeEditStatus(db, { schema, table, column }, false);
      }
    }
  }
}
