import type { TreeNode, NodeType } from "./types";

export const isTypedNode = <T extends NodeType>(
  node: TreeNode,
  type: T
): node is TreeNode<T> => {
  return node.type === type;
};
