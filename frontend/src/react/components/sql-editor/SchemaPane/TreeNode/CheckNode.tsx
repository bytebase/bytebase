import { useAppStore } from "@/react/stores/app";
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
  const target = (node as TreeNode<"check">).meta.target;
  const hasTable = "table" in target;
  const tableMetadata = useAppStore((s) =>
    hasTable
      ? s.getTableMetadata({
          database: target.database,
          schema: target.schema,
          table: target.table,
        })
      : undefined
  );
  const checkMetadata = hasTable
    ? tableMetadata?.checkConstraints.find((c) => c.name === target.check)
    : undefined;

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
