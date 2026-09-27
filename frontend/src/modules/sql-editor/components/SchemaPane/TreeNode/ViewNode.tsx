import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ViewIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function ViewNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"view">).meta.target;
  return (
    <CommonNode
      text={target.view}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<ViewIcon />}
    />
  );
}
