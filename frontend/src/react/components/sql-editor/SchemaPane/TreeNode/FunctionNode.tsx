import { useAppStore } from "@/react/stores/app";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { FunctionIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/** Replaces `TreeNode/FunctionNode.vue`. Same signature-vs-name fallback
 *  as ProcedureNode. */
export function FunctionNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"function">).meta.target;
  const schemaMetadata = useAppStore((s) =>
    s.getSchemaMetadata({ database: target.database, schema: target.schema })
  );
  const functionMetadata = schemaMetadata?.functions[target.position];

  return (
    <CommonNode
      text={(functionMetadata?.signature || functionMetadata?.name) ?? ""}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<FunctionIcon />}
    />
  );
}
