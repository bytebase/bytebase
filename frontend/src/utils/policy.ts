import { resolveCELExpr } from "@/plugins/cel";
import { hasFeature, useCurrentUserIamPolicy, usePolicyV1Store } from "@/store";
import { policyNamePrefix } from "@/store/modules/v1/common";
import type { ComposedDatabase, Instance } from "@/types";
import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { User } from "@/types/proto/v1/auth_service";
import {
  PolicyType,
  policyTypeToJSON,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV1 } from "./role";
import { extractEnvironmentNameListFromExpr } from "./v1";

export const isInstanceAccessible = (instance: Instance, user: User) => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-access-control",
      user.userRole
    )
  ) {
    // The current user has the super privilege to access all databases.
    // AKA. Owners and DBAs
    return true;
  }

  // See if the instance is in a production environment
  const { environment } = instance;
  if (environment.tier === "UNPROTECTED") {
    return true;
  }

  return false;
};

export const isDatabaseAccessible = (
  database: ComposedDatabase,
  user: User
) => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-access-control",
      user.userRole
    )
  ) {
    // The current user has the super privilege to access all databases.
    // AKA. Owners and DBAs
    return true;
  }

  if (hasFeature("bb.feature.access-control")) {
    const name = `${policyNamePrefix}${policyTypeToJSON(
      PolicyType.WORKSPACE_IAM
    )}`;
    const policy = usePolicyV1Store().getPolicyByName(name);
    if (policy) {
      const bindings = policy.workspaceIamPolicy?.bindings;
      if (bindings) {
        const querierBinding = bindings.find(
          (binding) => binding.role === "roles/QUERIER"
        );
        if (querierBinding) {
          const simpleExpr = resolveCELExpr(
            querierBinding.parsedExpr?.expr || Expr.fromPartial({})
          );
          const envNameList = extractEnvironmentNameListFromExpr(simpleExpr);
          if (envNameList.includes(database.instanceEntity.environment)) {
            return true;
          }
        }
      }
    }
  }

  const currentUserIamPolicy = useCurrentUserIamPolicy();
  if (currentUserIamPolicy.allowToQueryDatabaseV1(database)) {
    return true;
  }

  // denied otherwise
  return false;
};
