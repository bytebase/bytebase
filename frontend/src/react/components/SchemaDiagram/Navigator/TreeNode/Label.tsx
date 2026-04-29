import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import type { NavigatorTreeNode } from "../types";
import { isTypedNode } from "../utils";

interface LabelProps {
  node: NavigatorTreeNode;
  keyword: string;
}

/**
 * React port of `Navigator/TreeNode/Label.vue`. Uses the shared
 * `HighlightLabelText` for keyword highlighting (per the
 * use-highlightlabeltext memory rule — never inline highlight HTML).
 */
export function Label({ node, keyword }: LabelProps) {
  const { t } = useTranslation();
  const text = useMemo(() => {
    return node.data.name || t("db.schema.default");
  }, [node, t]);

  const id = `tree-node-label-${node.type}-${node.id}`;

  return (
    <div
      className="flex items-center gap-x-1 max-w-full"
      data-schema-editor-nav-tree-node-id={id}
    >
      <HighlightLabelText text={text} keyword={keyword} className="truncate" />
      {isTypedNode(node, "schema") && (
        <span className="shrink-0 text-control-light">
          ({node.data.tables.length})
        </span>
      )}
    </div>
  );
}
