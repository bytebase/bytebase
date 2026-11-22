import type { QueryDataPolicy } from "@/types/proto-es/v1/org_policy_service_pb";

export type EffectiveQueryDataPolicy = {
  disableCopyData: boolean;
  disableDownload: boolean;
  allowAdminDataSource: boolean;
  maximumResultRows: number;
};

const formatQueryDataPolicy = (
  policy: QueryDataPolicy | undefined
): EffectiveQueryDataPolicy => {
  const max = policy?.maximumResultRows ?? -1;
  return {
    disableCopyData: policy?.disableCopyData ?? false,
    disableDownload: policy?.disableDownload ?? false,
    allowAdminDataSource: policy?.allowAdminDataSource ?? false,
    maximumResultRows: max <= 0 ? Number.MAX_VALUE : max,
  };
};

export const getEffectiveQueryDataPolicy = (
  workspacePolicy?: QueryDataPolicy,
  projectPolicy?: QueryDataPolicy
): EffectiveQueryDataPolicy => {
  const workspace = formatQueryDataPolicy(workspacePolicy);
  const project = formatQueryDataPolicy(projectPolicy);
  return {
    ...workspace,
    maximumResultRows: Math.min(
      workspace.maximumResultRows,
      project.maximumResultRows
    ),
  };
};
