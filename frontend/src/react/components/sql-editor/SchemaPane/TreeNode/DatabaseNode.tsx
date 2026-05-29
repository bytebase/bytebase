import { EngineIcon } from "@/react/components/EngineIcon";
import { useAppDatabase } from "@/react/hooks/useAppDatabase";
import { extractDatabaseResourceName, getInstanceResource } from "@/utils";
import type { TreeNode } from "../schemaTree";
import { CommonNode } from "./CommonNode";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/DatabaseNode.vue`. The schema tree's root node:
 * engine icon + database display name (highlight-aware).
 */
export function DatabaseNode({ node, keyword }: Props) {
  const target = (node as TreeNode<"database">).meta.target;

  const database = useAppDatabase(target.database);

  const databaseName = extractDatabaseResourceName(database.name).databaseName;
  const instance = getInstanceResource(database);

  return (
    <CommonNode
      text={databaseName}
      keyword={keyword}
      highlight={true}
      icon={<EngineIcon engine={instance.engine} className="size-4" />}
    />
  );
}
