import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import type { SQLEditorTreeNode } from "@/types";

type Props = {
  readonly node: SQLEditorTreeNode;
  readonly keyword: string;
};

/** Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/TreeNode/LabelNode.vue. */
export function LabelNode({ node, keyword }: Props) {
  const { t } = useTranslation();
  const label = (node as SQLEditorTreeNode<"label">).meta.target;

  return (
    <div className="truncate">
      <span>{label.key}</span>
      <span className="mr-0.5">:</span>
      {label.value ? (
        <HighlightLabelText text={label.value} keyword={keyword} />
      ) : (
        <span className="text-control-placeholder">
          {t("label.empty-label-value")}
        </span>
      )}
    </div>
  );
}
