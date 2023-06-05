import { hasFeature, useCurrentUserIamPolicy } from "@/store";
import type { ComposedDatabase, Instance } from "@/types";
import { hasWorkspacePermissionV1 } from "./role";
import { Policy, PolicyType } from "@/types/proto/v1/org_policy_service";
import { User } from "@/types/proto/v1/auth_service";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";

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
  policyList: Policy[],
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
    const { environmentEntity } = database.instanceEntity;
    if (environmentEntity.tier === EnvironmentTier.PROTECTED) {
      const policy = policyList.find((policy) => {
        const { type, resourceUid, enforce } = policy;
        return (
          type === PolicyType.ACCESS_CONTROL &&
          resourceUid === `${database.uid}` &&
          enforce
        );
      });
      if (policy) {
        // The database is in the allowed list
        return true;
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
