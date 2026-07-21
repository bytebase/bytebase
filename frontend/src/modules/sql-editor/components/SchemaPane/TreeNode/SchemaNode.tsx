import { Box } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/SchemaNode.vue`. Schema namespace node — falls
 * back to the i18n "default" label when the schema name is empty.
 */
export function SchemaNode({ node, keyword }: Props) {
  const { t } = useTranslation();
  const target = (node as TreeNode<"schema">).meta.target;

  return (
    <CommonNode
      text={target.schema}
      keyword={keyword}
      highlight={true}
      icon={<Box className="size-4" />}
      fallbackText={<>{t("db.schema.default")}</>}
    />
  );
}
