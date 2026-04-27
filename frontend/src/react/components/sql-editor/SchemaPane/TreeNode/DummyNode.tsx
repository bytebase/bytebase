import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import type { TreeNode } from "../schemaTree";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/DummyNode.vue`. Renders an "<Empty>" / "<Error>"
 * placeholder span; if the underlying target carries an `error`, the
 * span exposes that message via Tooltip on hover.
 */
export function DummyNode({ node }: Props) {
  const { t } = useTranslation();
  const error = (node as TreeNode<"error">).meta.target.error;
  const text = `<${error ? t("common.error") : t("common.empty")}>`;
  const span = (
    <span className="text-control-placeholder ml-[2px]">{text}</span>
  );

  if (!error) return span;

  return (
    <Tooltip
      content={
        <div className="text-error wrap-break-word break-all">
          {String(error)}
        </div>
      }
    >
      {span}
    </Tooltip>
  );
}
