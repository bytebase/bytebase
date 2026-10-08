import { FocusButton } from "../../common/FocusButton";
import type { NavigatorTreeNode } from "../types";
import { isTypedNode } from "../utils";

interface SuffixProps {
  node: NavigatorTreeNode;
}

export function Suffix({ node }: SuffixProps) {
  if (isTypedNode(node, "table")) {
    // `FocusButton` defaults to `invisible` for unfocused tables (the
    // canvas surface only shows it on per-table hover). Reveal it on row
    // hover with `group-hover:visible` — the surrounding tree row in
    // `Tree.tsx` sets `group`.
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
