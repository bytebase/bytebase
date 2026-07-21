import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { TableLeafIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function TableNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"table">).meta.target;
  return (
    <CommonNode
      text={target.table}
      keyword={keyword}
      highlight={true}
      icon={<TableLeafIcon />}
    />
  );
}
