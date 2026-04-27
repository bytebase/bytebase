import { EngineIconPath } from "@/components/InstanceForm/constants";
import { useVueState } from "@/react/hooks/useVueState";
import { useDatabaseV1Store } from "@/store";
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
  const databaseStore = useDatabaseV1Store();
  const target = (node as TreeNode<"database">).meta.target;

  const database = useVueState(() =>
    databaseStore.getDatabaseByName(target.database)
  );

  const databaseName = extractDatabaseResourceName(database.name).databaseName;
  const instance = getInstanceResource(database);
  const iconPath = EngineIconPath[instance.engine];

  return (
    <CommonNode
      text={databaseName}
      keyword={keyword}
      highlight={true}
      icon={iconPath ? <img src={iconPath} alt="" className="size-4" /> : null}
    />
  );
}
