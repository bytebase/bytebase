import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { PackageIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

export function PackageNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"package">).meta.target;
  return (
    <CommonNode
      text={target.package}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<PackageIcon />}
    />
  );
}
