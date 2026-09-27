import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { TriggerIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function TriggerNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"trigger">).meta.target;
  return (
    <CommonNode
      text={target.trigger}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<TriggerIcon />}
    />
  );
}
