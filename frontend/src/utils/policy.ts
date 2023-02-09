import { hasFeature } from "@/store";
import type { Database, Instance, Policy, Principal } from "@/types";
import { hasWorkspacePermission } from "./role";

export const isInstanceAccessible = (instance: Instance, user: Principal) => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-access-control",
      user.role
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
  database: Database,
  policyList: Policy[],
  user: Principal
) => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-access-control",
      user.role
    )
  ) {
    // The current user has the super privilege to access all databases.
    // AKA. Owners and DBAs
    return true;
  }

  const { environment } = database.instance;
  if (environment.tier === "UNPROTECTED") {
    return true;
  }

  const policy = policyList.find((policy) => {
    const { type, resourceId, rowStatus } = policy;
    return (
      type === "bb.policy.access-control" &&
      resourceId === database.id &&
      rowStatus === "NORMAL"
    );
  });
  if (policy) {
    // The database is in the allowed list
    return true;
  }
  // denied otherwise
  return false;
};
