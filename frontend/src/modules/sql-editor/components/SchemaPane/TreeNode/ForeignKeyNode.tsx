import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ForeignKeyIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function ForeignKeyNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"foreign-key">).meta.target;
  return (
    <CommonNode
      text={target.foreignKey}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<ForeignKeyIcon />}
    />
  );
}
