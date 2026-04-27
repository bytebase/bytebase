import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { SequenceIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function SequenceNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"sequence">).meta.target;
  return (
    <CommonNode
      text={target.sequence}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<SequenceIcon />}
    />
  );
}
