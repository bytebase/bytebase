import { FocusButton } from "../../common/FocusButton";
import type { NavigatorTreeNode } from "../types";
import { isTypedNode } from "../utils";

interface SuffixProps {
  node: NavigatorTreeNode;
}

/** React port of `Navigator/TreeNode/Suffix.vue`. */
export function Suffix({ node }: SuffixProps) {
  if (isTypedNode(node, "table")) {
    // `FocusButton` defaults to `invisible` for unfocused tables (the
    // canvas surface only shows it on per-table hover). The Vue tree
    // relied on a `.bb-schema-diagram-nav-tree .n-tree-node:hover
    // .focus-btn { visibility: visible }` rule that didn't survive the
    // port. Re-create the affordance with `group-hover:visible` — the
    // surrounding tree row in `Tree.tsx` sets `group`.
    return (
      <FocusButton
        table={node.data}
        setCenter
        className="group-hover:visible"
      />
    );
  }
  return null;
}
