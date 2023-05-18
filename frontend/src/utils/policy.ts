import { hasFeature, useCurrentUserIamPolicy } from "@/store";
import type { Database, Instance } from "@/types";
import { hasWorkspacePermissionV1 } from "./role";
import { Policy, PolicyType } from "@/types/proto/v1/org_policy_service";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { User } from "@/types/proto/v1/auth_service";

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
  if (environment.tier === EnvironmentTier.UNPROTECTED) {
    return true;
  }

  return false;
};

export const isDatabaseAccessible = (
  database: Database,
  policyList: Policy[],
  user: User
) => {
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

  const { environment } = database.instance;
  if (environment.tier === EnvironmentTier.UNPROTECTED) {
    return true;
  }

  const policy = policyList.find((policy) => {
    const { type, resourceUid, enforce } = policy;
    return (
      type === PolicyType.ACCESS_CONTROL &&
      resourceUid === `${database.id}` &&
      enforce
    );
  });
  if (policy) {
    // The database is in the allowed list
    return true;
  }
  const currentUserIamPolicy = useCurrentUserIamPolicy();
  if (currentUserIamPolicy.allowToQueryDatabase(database)) {
    return true;
  }

  // denied otherwise
  return false;
};
