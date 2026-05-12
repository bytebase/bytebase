import { useMemo } from "react";
import { useDBSchemaV1Store } from "@/store";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ColumnIcon, IndexIcon, PrimaryKeyIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

// Per-table caches: each entry is a `Map<columnName, ColumnMetadata>` keyed
// by the table-like metadata object that owns the columns. Two reasons for
// the WeakMap-of-Maps pattern:
//   1. The previous implementation did `tableMetadata.columns.find(...)`
//      from every ColumnNode, which is O(columns-per-table) per row. For a
//      200-column table with ~30 virtualized rows, that's 6,000 reactive
//      proxy reads per expand — and every read goes through Vue's tracker
//      because the metadata is a Pinia/Vue reactive proxy.
//   2. WeakMap keys (the parent metadata reference) drop their cache entry
//      automatically when the metadata refreshes — no manual invalidation.
const tableColumnIndex = new WeakMap<object, Map<string, ColumnMetadata>>();
function findColumn(parent: { columns: ColumnMetadata[] }, name: string) {
  let map = tableColumnIndex.get(parent);
  if (!map) {
    map = new Map(parent.columns.map((c) => [c.name, c]));
    tableColumnIndex.set(parent, map);
  }
  return map.get(name);
}

/**
 * Replaces `TreeNode/ColumnNode.vue`. The icon and trailing type-tag both
 * depend on the column's resolved metadata:
 *  - PrimaryKeyIcon when the column is part of the primary index.
 *  - IndexIcon when it's part of any other index (and not the PK).
 *  - ColumnIcon otherwise.
 *  - The trailing type-tag (`int`, `varchar(255)`, …) is rendered when
 *    the metadata fetch has resolved.
 *
 * Lookups use `useMemo` (not `useVueState`). The schema tree is rebuilt
 * by `SchemaPane` whenever the database metadata changes, which produces
 * fresh `target` objects and remounts these rows with current metadata —
 * a Vue watch per ColumnNode is unnecessary subscription overhead, and
 * `flush: "sync"` watch setup was the dominant cost of expanding a
 * Columns folder with hundreds of children.
 */
export function ColumnNode({ node, keyword }: Props) {
  const dbSchema = useDBSchemaV1Store();
  const target = (node as TreeNode<"column">).meta.target;

  const tableMetadata = useMemo(() => {
    if ("table" in target) {
      const { database, schema, table } = target;
      return dbSchema.getTableMetadata({ database, schema, table });
    }
    return undefined;
  }, [target, dbSchema]);

  const columnMetadata = useMemo(() => {
    const { database, schema, column } = target;
    if ("table" in target && tableMetadata) {
      return findColumn(tableMetadata, column);
    }
    if ("externalTable" in target) {
      const schemaMetadata = dbSchema.getSchemaMetadata({ database, schema });
      const ext = schemaMetadata?.externalTables.find(
        (t) => t.name === target.externalTable
      );
      return ext ? findColumn(ext, column) : undefined;
    }
    if ("view" in target) {
      const schemaMetadata = dbSchema.getSchemaMetadata({ database, schema });
      const view = schemaMetadata?.views.find((v) => v.name === target.view);
      return view ? findColumn(view, column) : undefined;
    }
    return undefined;
  }, [target, tableMetadata, dbSchema]);

  const { isPrimaryKey, isIndex } = useMemo(() => {
    if (!("table" in target) || !tableMetadata) {
      return { isPrimaryKey: false, isIndex: false };
    }
    const { column } = target;
    const pk = tableMetadata.indexes.find((idx) => idx.primary);
    const inPk = pk ? pk.expressions.includes(column) : false;
    if (inPk) return { isPrimaryKey: true, isIndex: false };
    const inIdx = tableMetadata.indexes.some((idx) =>
      idx.expressions.includes(column)
    );
    return { isPrimaryKey: false, isIndex: inIdx };
  }, [target, tableMetadata]);

  const icon = isPrimaryKey ? (
    <PrimaryKeyIcon />
  ) : isIndex ? (
    <IndexIcon className="size-4! text-accent/80" />
  ) : (
    <ColumnIcon />
  );

  return (
    <CommonNode
      text={target.column}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={icon}
      suffix={
        columnMetadata?.type ? (
          <div className="flex items-center justify-end gap-1 overflow-hidden whitespace-nowrap shrink opacity-80 font-normal!">
            <span className="inline-flex items-center rounded-sm border border-control-border bg-control-bg/40 text-control h-4 px-[3px] text-[10px] leading-none">
              {columnMetadata.type}
            </span>
          </div>
        ) : null
      }
    />
  );
}
