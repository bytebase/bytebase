import { uniq } from "lodash-es";
import { extractUserId, useGroupStore, useWorkspaceV1Store } from "@/store";
import { roleNamePrefix, userNamePrefix } from "@/store/modules/v1/common";
import {
  PresetRoleType,
  groupBindingPrefix,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import type { IamPolicy, Binding } from "@/types/proto/v1/iam_policy";
import { convertFromExpr } from "@/utils/issue/cel";

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

// getUserEmailListInBinding will extract users in the IAM policy binding.
// If the binding is group, will conains all members email in the group.
// It can also includs ALL_USERS_USER_EMAIL
export const getUserEmailListInBinding = ({
  binding,
  ignoreGroup,
}: {
  binding: Binding;
  ignoreGroup: boolean;
}): string[] => {
  if (isBindingPolicyExpired(binding)) {
    return [];
  }

  const groupStore = useGroupStore();
  const emailList = [];

  for (const member of binding.members) {
    if (member.startsWith(groupBindingPrefix)) {
      if (ignoreGroup) {
        continue;
      }
      const group = groupStore.getGroupByIdentifier(member);
      if (!group) {
        continue;
      }

      emailList.push(...group.members.map((m) => extractUserId(m.member)));
    } else {
      const email = extractUserId(member);
      // ATTENTION:
      // the email can be ALL_USERS_USER_EMAIL
      emailList.push(email);
    }
  }
  return uniq(emailList);
};

// memberMapToRolesInProjectIAM return the Map<users/{email}, Set<roles/{role}>>
// the user could includes users/ALL_USERS_USER_EMAIL
export const memberMapToRolesInProjectIAM = (
  iamPolicy: IamPolicy,
  targetRole?: string
): Map<string, Set<string>> => {
  const workspaceStore = useWorkspaceV1Store();
  // Map<users/{email}, Set<roles/{role}>>
  const rolesMapByName = new Map<string, Set<string>>();

  // Handle project level roles.
  for (const binding of iamPolicy.bindings) {
    if (targetRole && binding.role !== targetRole) {
      continue;
    }
    if (isBindingPolicyExpired(binding)) {
      continue;
    }

    const emails = getUserEmailListInBinding({ binding, ignoreGroup: false });

    for (const email of emails) {
      const key = `${userNamePrefix}${email}`;
      if (!rolesMapByName.has(key)) {
        rolesMapByName.set(key, new Set());
      }
      rolesMapByName.get(key)?.add(binding.role);
    }
  }

  // Handle workspace level project roles.
  for (const [role, userSet] of workspaceStore.roleMapToUsers.entries()) {
    if (PRESET_WORKSPACE_ROLES.includes(role)) {
      continue;
    }
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

export const roleListInIAM = ({
  policy,
  email,
  ignoreGroup,
}: {
  policy: IamPolicy;
  email: string;
  ignoreGroup: boolean;
}) => {
  const roles = policy.bindings
    .filter((binding) => {
      if (binding.role === PresetRoleType.WORKSPACE_MEMBER) {
        return false;
      }
      if (isBindingPolicyExpired(binding)) {
        return false;
      }
      const emailList = getUserEmailListInBinding({ binding, ignoreGroup });
      return emailList.includes(email);
    })
    .map((binding) => binding.role);

  if (!roles.some((role) => role.startsWith(`${roleNamePrefix}workspace`))) {
    roles.push(PresetRoleType.WORKSPACE_MEMBER);
  }

  return roles;
};
