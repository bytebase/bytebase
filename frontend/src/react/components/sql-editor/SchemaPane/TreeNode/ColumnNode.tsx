import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ColumnIcon, IndexIcon, PrimaryKeyIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/ColumnNode.vue`. The icon and trailing type-tag both
 * depend on the column's resolved metadata:
 *  - PrimaryKeyIcon when the column is part of the primary index.
 *  - IndexIcon when it's part of any other index (and not the PK).
 *  - ColumnIcon otherwise.
 *  - The trailing type-tag (`int`, `varchar(255)`, …) is rendered when
 *    the metadata fetch has resolved.
 *
 * Reads from `dbSchemaStore.getTableMetadata` / `getSchemaMetadata`
 * through `useVueState` so cache hydration triggers a re-render.
 */
export function ColumnNode({ node, keyword }: Props) {
  const dbSchema = useDBSchemaV1Store();
  const target = (node as TreeNode<"column">).meta.target;

  const tableMetadata = useVueState(() => {
    if ("table" in target) {
      const { database, schema, table } = target;
      return dbSchema.getTableMetadata({ database, schema, table });
    }
    return undefined;
  });

  const columnMetadata = useVueState(() => {
    const { database, schema, column } = target;
    const schemaMetadata = dbSchema.getSchemaMetadata({ database, schema });

    if ("table" in target) {
      return tableMetadata?.columns.find((c) => c.name === column);
    } else if ("externalTable" in target) {
      return schemaMetadata?.externalTables
        .find((t) => t.name === target.externalTable)
        ?.columns.find((c) => c.name === column);
    } else if ("view" in target) {
      return schemaMetadata?.views
        .find((v) => v.name === target.view)
        ?.columns.find((c) => c.name === column);
    }
    return undefined;
  });

  let isPrimaryKey = false;
  let isIndex = false;
  if ("table" in target) {
    const { column } = target;
    const pk = tableMetadata?.indexes.find((idx) => idx.primary);
    isPrimaryKey = pk ? pk.expressions.includes(column) : false;
    if (!isPrimaryKey) {
      isIndex = !!tableMetadata?.indexes.some((idx) =>
        idx.expressions.includes(column)
      );
    }
  }

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
