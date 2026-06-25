import { isUndefined, uniq } from "lodash-es";
import { getProjectByName } from "@/react/stores/app/projectAccess";
import {
  ensureGroupIdentifier,
  extractServiceAccountId,
  extractWorkloadIdentityId,
  groupNamePrefix,
} from "@/store";
import {
  getUserFullNameByType,
  serviceAccountNamePrefix,
  userNamePrefix,
  workloadIdentityNamePrefix,
} from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  groupBindingPrefix,
  type QueryPermission,
  QueryPermissionQueryAny,
  serviceAccountBindingPrefix,
  unknownUser,
  workloadIdentityBindingPrefix,
} from "@/types";
import type { Expr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { Binding, IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { appStoreUtilBridge } from "@/utils/app-store-bridge";
import { convertFromExpr } from "@/utils/issue/cel";
import { ensureUserFullName } from "@/utils/v1/user";

export const isBindingPolicyExpired = (binding: Binding): boolean => {
  if (binding.parsedExpr) {
    const conditionExpr = convertFromExpr(binding.parsedExpr);
    if (conditionExpr.expiredTime) {
      const expiration = new Date(conditionExpr.expiredTime);
      if (expiration < new Date()) {
        return true;
      }
    }
  }
  return false;
};

// Revoke a single member from one specific role binding, returning a new policy
// (the input is left untouched).
//
// The target binding is resolved by reference identity first, then by role +
// condition expression + membership. Identity matching matters because a policy
// can hold several bindings that share the same (role, condition) — e.g. when
// multiple users are granted the same database with no expiration, each grant is
// stored as a separate single-member binding. Resolving only by (role,
// condition) would match the first such binding, which need not contain the
// member being revoked, silently leaving the intended grant in place.
export const revokeMemberFromBinding = (
  policy: IamPolicy,
  targetBinding: Binding,
  member: string
): IamPolicy => {
  let index = policy.bindings.indexOf(targetBinding);
  if (index < 0) {
    index = policy.bindings.findIndex(
      (b) =>
        b.role === targetBinding.role &&
        (b.condition?.expression ?? "") ===
          (targetBinding.condition?.expression ?? "") &&
        b.members.includes(member)
    );
  }
  const next = structuredClone(policy);
  if (index >= 0) {
    next.bindings[index].members = next.bindings[index].members.filter(
      (m) => m !== member
    );
  }
  next.bindings = next.bindings.filter((b) => b.members.length > 0);
  return next;
};

export const convertMemberToFullname = (member: string) => {
  if (member.startsWith(groupBindingPrefix)) {
    return ensureGroupIdentifier(member);
  } else if (member.startsWith(serviceAccountBindingPrefix)) {
    const email = extractServiceAccountId(member);
    return `${serviceAccountNamePrefix}${email}`;
  } else if (member.startsWith(workloadIdentityBindingPrefix)) {
    const email = extractWorkloadIdentityId(member);
    return `${workloadIdentityNamePrefix}${email}`;
  } else {
    // ATTENTION: the email can be ALL_USERS_USER_EMAIL
    return ensureUserFullName(member);
  }
};

// getUserListInBinding will extract users in the IAM policy binding.
// If the binding is group, will conains all members in the group.
// The return value should be the user full name with the prefix:
// - users/{email}, could includs ALL_USERS_USER_EMAIL
// - serviceAccounts/{email}
// - workloadIdentities/{email}
export const getUserListInBinding = ({
  binding,
  ignoreGroup,
  getGroupByIdentifier,
}: {
  binding: Binding;
  ignoreGroup: boolean;
  // Resolves a group from its binding member string. Defaults to the Pinia
  // group store; React callers pass a resolver backed by the app store so
  // group expansion reads from the same cache they populate.
  getGroupByIdentifier?: (identifier: string) => Group | undefined;
}): string[] => {
  if (isBindingPolicyExpired(binding)) {
    return [];
  }

  const resolveGroup =
    getGroupByIdentifier ??
    ((identifier: string) =>
      appStoreUtilBridge()?.getGroupByIdentifier(identifier));
  const fullnameList = [];

  for (const member of binding.members) {
    const fullname = convertMemberToFullname(member);
    if (fullname.startsWith(groupNamePrefix)) {
      if (ignoreGroup) {
        continue;
      }
      const group = resolveGroup(member);
      if (!group) {
        continue;
      }
      for (const groupMember of group.members) {
        // the group member MUST be human.
        fullnameList.push(ensureUserFullName(groupMember.member));
      }
    } else {
      fullnameList.push(fullname);
    }
  }
  return uniq(fullnameList);
};

// memberMapToRolesInProjectIAM return the Map<users/{email}, Set<roles/{role}>>
// the user could includes users/ALL_USERS_USER_EMAIL
export const memberMapToRolesInProjectIAM = (
  iamPolicy: IamPolicy,
  targetRole?: string,
  getGroupByIdentifier?: (identifier: string) => Group | undefined
): Map<string, Set<string>> => {
  // Map<userfullname, Set<roles/{role}>>
  const rolesMapByName = new Map<string, Set<string>>();

  // Handle project level roles.
  for (const binding of iamPolicy.bindings) {
    if (targetRole && binding.role !== targetRole) {
      continue;
    }
    if (isBindingPolicyExpired(binding)) {
      continue;
    }

    const fullnames = getUserListInBinding({
      binding,
      ignoreGroup: false,
      getGroupByIdentifier,
    });
    for (const fullname of fullnames) {
      if (!rolesMapByName.has(fullname)) {
        rolesMapByName.set(fullname, new Set());
      }
      rolesMapByName.get(fullname)?.add(binding.role);
    }
  }

  // Handle workspace level project roles.
  const roleMapToUsers =
    appStoreUtilBridge()?.workspaceRoleMapToUsers() ??
    new Map<string, Set<string>>();
  for (const [role, userSet] of roleMapToUsers.entries()) {
    if (targetRole && role !== targetRole) {
      continue;
    }
    for (const user of userSet.values()) {
      if (!rolesMapByName.has(user)) {
        rolesMapByName.set(user, new Set());
      }
      rolesMapByName.get(user)?.add(role);
    }
  }

  return rolesMapByName;
};

export const filterBindingsByUserName = ({
  policy,
  name,
  ignoreGroup,
}: {
  policy: IamPolicy;
  name: string; // the name should be the fullname
  ignoreGroup: boolean;
}): Binding[] => {
  return policy.bindings.filter((binding) => {
    if (isBindingPolicyExpired(binding)) {
      return false;
    }
    const fullnameList = getUserListInBinding({ binding, ignoreGroup });
    return (
      fullnameList.includes(`${userNamePrefix}${ALL_USERS_USER_EMAIL}`) ||
      fullnameList.includes(name)
    );
  });
};

// Project-level IAM permission check. Reads the React app store (project IAM
// policy + roles) via the util bridge — relocated from the deleted Pinia
// `projectIamPolicy` store, whose data was never populated in the React shell.
const checkProjectIAMPolicyWithExpr = (
  user: User,
  project: Project,
  requiredPermissions: QueryPermission[],
  bindingExprCheck: (expr?: Expr) => boolean
): boolean => {
  const policy = appStoreUtilBridge()?.getProjectIamPolicy(project.name);
  if (!policy) {
    return false;
  }
  for (const binding of policy.bindings) {
    const nameList = getUserListInBinding({ binding, ignoreGroup: false });
    if (
      !nameList.includes(getUserFullNameByType(user)) &&
      !nameList.includes(`${userNamePrefix}${ALL_USERS_USER_EMAIL}`)
    ) {
      continue;
    }
    const permissions =
      appStoreUtilBridge()?.getRoleByName(binding.role)?.permissions || [];
    for (const permission of permissions) {
      if (requiredPermissions.includes(permission as QueryPermission)) {
        if (bindingExprCheck(binding.parsedExpr)) {
          return true;
        }
      }
    }
  }
  return false;
};

export const checkQuerierPermission = (
  database: Database,
  permissions: QueryPermission[] = QueryPermissionQueryAny,
  schema?: string,
  table?: string
): boolean => {
  return checkProjectIAMPolicyWithExpr(
    appStoreUtilBridge()?.currentUser() ?? unknownUser(),
    getProjectByName(database.project),
    permissions,
    (expr?: Expr): boolean => {
      if (!expr) {
        return true;
      }
      const conditionExpr = convertFromExpr(expr);
      if (
        conditionExpr.expiredTime &&
        new Date(conditionExpr.expiredTime).getTime() < Date.now()
      ) {
        return false;
      }
      if (
        conditionExpr.databaseResources &&
        conditionExpr.databaseResources.length > 0
      ) {
        for (const databaseResource of conditionExpr.databaseResources) {
          if (databaseResource.databaseFullName === database.name) {
            if (isUndefined(schema) && isUndefined(table)) {
              return true;
            }
            if (
              isUndefined(databaseResource.schema) ||
              (isUndefined(databaseResource.schema) &&
                isUndefined(databaseResource.table)) ||
              (databaseResource.schema === schema &&
                databaseResource.table === table)
            ) {
              return true;
            }
          }
        }
        return false;
      }
      return true;
    }
  );
};
