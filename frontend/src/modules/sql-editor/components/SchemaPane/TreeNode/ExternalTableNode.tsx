import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ExternalTableIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function ExternalTableNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"external-table">).meta.target;
  return (
    <CommonNode
      text={target.externalTable}
      keyword={keyword}
      highlight={true}
      icon={<ExternalTableIcon />}
    />
  );
}
