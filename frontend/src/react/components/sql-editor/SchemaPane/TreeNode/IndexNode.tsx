import { useAppStore } from "@/react/stores/app";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { IndexIcon, PrimaryKeyIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/IndexNode.vue`. PrimaryKeyIcon when the index is
 * marked `primary`, IndexIcon otherwise.
 */
export function IndexNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"index">).meta.target;
  const tableMetadata = useAppStore((s) =>
    s.getTableMetadata({
      database: target.database,
      schema: target.schema,
      table: target.table,
    })
  );
  const isPrimaryKey = !!tableMetadata.indexes.find(
    (i) => i.name === target.index
  )?.primary;

  return (
    <CommonNode
      text={target.index}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={
        isPrimaryKey ? (
          <PrimaryKeyIcon />
        ) : (
          <IndexIcon className="size-4! text-accent/80" />
        )
      }
    />
  );
}
