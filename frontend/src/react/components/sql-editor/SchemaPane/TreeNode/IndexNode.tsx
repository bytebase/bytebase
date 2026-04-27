import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
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
  const dbSchema = useDBSchemaV1Store();
  const target = (node as TreeNode<"index">).meta.target;

  const isPrimaryKey = useVueState(() => {
    const { database, schema, table, index } = target;
    const tableMetadata = dbSchema.getTableMetadata({
      database,
      schema,
      table,
    });
    return !!tableMetadata.indexes.find((i) => i.name === index)?.primary;
  });

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
