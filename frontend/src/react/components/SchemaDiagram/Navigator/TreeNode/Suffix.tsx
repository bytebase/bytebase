import { FocusButton } from "../../common/FocusButton";
import type { NavigatorTreeNode } from "../types";
import { isTypedNode } from "../utils";

interface SuffixProps {
  node: NavigatorTreeNode;
}

/** React port of `Navigator/TreeNode/Suffix.vue`. */
export function Suffix({ node }: SuffixProps) {
  if (isTypedNode(node, "table")) {
    return <FocusButton table={node.data} setCenter className="focus-btn" />;
  }
  return null;
}
