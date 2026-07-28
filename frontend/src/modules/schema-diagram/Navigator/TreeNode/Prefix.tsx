import { Layers, Table2 } from "lucide-react";
import type { NavigatorTreeNode } from "../types";

interface PrefixProps {
  node: NavigatorTreeNode;
}

/** React port of `Navigator/TreeNode/Prefix.vue`. */
export function Prefix({ node }: PrefixProps) {
  if (node.type === "schema") return <Layers className="size-4" />;
  if (node.type === "table") return <Table2 className="size-4" />;
  return null;
}
