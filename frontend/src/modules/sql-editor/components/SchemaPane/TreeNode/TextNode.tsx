import { Folder } from "lucide-react";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Folder-style row used for "Tables / Views / Indexes / …" headers. If the
 * target sets a `render` ReactNode, we pass it through CommonNode's
 * `children` to override the default text+icon layout entirely.
 */
export function TextNode({ node }: Props) {
  const target = (node as TreeNode<"expandable-text">).meta.target;
  const text = typeof target.text === "function" ? target.text() : target.text;

  if (target.render) {
    return <CommonNode text={text}>{target.render}</CommonNode>;
  }

  return (
    <CommonNode
      text={text}
      highlight={false}
      data-mock-type={target.mockType}
      icon={<Folder className="size-4 stroke-accent fill-accent/10" />}
    />
  );
}
