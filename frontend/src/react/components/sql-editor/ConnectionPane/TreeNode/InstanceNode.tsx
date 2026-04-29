import { EngineIcon } from "@/react/components/EngineIcon";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import type { SQLEditorTreeNode } from "@/types";
import { instanceV1Name } from "@/utils";

type Props = {
  readonly node: SQLEditorTreeNode;
  readonly keyword: string;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/TreeNode/InstanceNode.vue.
 * The Vue version also supported an "environment prefix" path gated on
 * `hasEnvironmentContext`, which was hard-coded to `true` — so in practice
 * only the instance name + engine icon was ever rendered. We mirror that
 * reduced behavior.
 */
export function InstanceNode({ node, keyword }: Props) {
  const target = (node as SQLEditorTreeNode<"instance">).meta.target;
  const instance = {
    ...target,
    $typeName: "bytebase.v1.InstanceResource" as const,
  };
  return (
    <div className="flex items-center max-w-full overflow-hidden gap-x-1">
      <span className="inline-flex items-center gap-x-1">
        <EngineIcon engine={instance.engine} className="size-4" />
        <span className="truncate">
          <HighlightLabelText
            text={instanceV1Name(instance)}
            keyword={keyword}
          />
        </span>
      </span>
    </div>
  );
}
