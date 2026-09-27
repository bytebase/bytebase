import type { NavigatorTreeNode, NodeType } from "./types";

export const isTypedNode = <T extends NodeType>(
  node: NavigatorTreeNode,
  type: T
): node is NavigatorTreeNode<T> => {
  return node.type === type;
};
