import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ProcedureIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/ProcedureNode.vue`. Reads procedure metadata to
 * prefer the resolved `signature` over the bare `name` (engines like
 * Oracle expose overloads where signature is the disambiguator).
 */
export function ProcedureNode({ node, keyword }: Props) {
  const dbSchema = useDBSchemaV1Store();
  const target = (node as TreeNode<"procedure">).meta.target;

  const procedureMetadata = useVueState(
    () =>
      dbSchema.getSchemaMetadata({
        database: target.database,
        schema: target.schema,
      })?.procedures[target.position]
  );

  return (
    <CommonNode
      text={(procedureMetadata?.signature || procedureMetadata?.name) ?? ""}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<ProcedureIcon />}
    />
  );
}
