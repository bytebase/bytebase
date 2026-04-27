import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";
import { ColumnIcon } from "./icons";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/DependencyColumnNode.vue`. Renders a view's
 * upstream column in `[schema.]table.column` form.
 */
export function DependencyColumnNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"dependency-column">).meta.target;
  const { schema, table, column } = target.dependencyColumn;
  const parts = [table, column];
  if (schema) parts.unshift(schema);
  const text = parts.join(".");

  return (
    <CommonNode
      text={text}
      keyword={keyword}
      highlight={true}
      indent={0}
      icon={<ColumnIcon />}
    />
  );
}
