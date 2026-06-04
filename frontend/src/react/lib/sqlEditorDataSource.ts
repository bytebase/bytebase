import { head } from "lodash-es";
import { useAppStore } from "@/react/stores/app";
import type { QueryDataSourceType } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { getInstanceResource } from "@/utils/v1/database";

// Picks the data-source id for a query, honoring the workspace DATA_QUERY
// policy's `allowAdminDataSource`. Reads the React app store directly (the
// legacy `@/utils/sqlEditor` version used the Pinia `useQueryDataPolicy`
// composable, whose store is not populated in the React shell — so admin data
// sources were never selectable). `allowAdminDataSource` is a workspace-level
// flag (the legacy policy getter took it from the workspace policy only).
export const getValidDataSourceByPolicy = async (
  database: Database,
  type?: QueryDataSourceType
): Promise<string> => {
  const instanceResource = getInstanceResource(database);
  const adminDataSource = instanceResource.dataSources.find(
    (ds) => ds.type === DataSourceType.ADMIN
  )!;
  const readonlyDataSources = instanceResource.dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );

  const store = useAppStore.getState();
  const workspace = store.workspaceResourceName();
  await store.getOrFetchPolicyByParentAndType({
    parentPath: workspace,
    policyType: PolicyType.DATA_QUERY,
  });
  const { allowAdminDataSource } = store.getQueryDataPolicyByParent(workspace);

  if (allowAdminDataSource && type === DataSourceType.ADMIN) {
    return adminDataSource.id;
  }
  return head(readonlyDataSources)?.id ?? adminDataSource.id;
};
