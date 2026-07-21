import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { TablePartitionIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function PartitionTableNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"partition-table">).meta.target;
  return (
    <CommonNode
      text={target.partition}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<TablePartitionIcon />}
    />
  );
}
