import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { CheckConstraintIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/CheckNode.vue`. Renders the constraint name plus a
 * trailing tag with the resolved check expression (if metadata is loaded).
 */
export function CheckNode({ node, keyword }: Props) {
  const dbSchema = useDBSchemaV1Store();
  const target = (node as TreeNode<"check">).meta.target;

  const checkMetadata = useVueState(() => {
    if ("table" in target) {
      const { database, schema, table, check } = target;
      const tableMetadata = dbSchema.getTableMetadata({
        database,
        schema,
        table,
      });
      return tableMetadata?.checkConstraints.find((c) => c.name === check);
    }
    return undefined;
  });

  return (
    <CommonNode
      text={target.check}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<CheckConstraintIcon />}
      suffix={
        checkMetadata?.expression ? (
          <div className="flex items-center justify-end gap-1 overflow-hidden whitespace-nowrap shrink opacity-80 font-normal!">
            <span className="inline-flex items-center rounded-sm border border-control-border bg-control-bg/40 text-control h-4 px-[3px] text-[10px] leading-none">
              {checkMetadata.expression}
            </span>
          </div>
        ) : null
      }
    />
  );
}
