import { orderBy, uniq } from "lodash-es";
import { extractUserEmail, useUserStore, useGroupStore } from "@/store";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  PresetRoleType,
  groupBindingPrefix,
  PRESET_WORKSPACE_ROLES,
  type ComposedUser,
  unknownUser,
} from "@/types";
import { getUserEmailInBinding } from "@/types";
import type { IamPolicy, Binding } from "@/types/proto/v1/iam_policy";
import { convertFromExpr } from "@/utils/issue/cel";

export const isBindingPolicyExpired = (binding: Binding): boolean => {
  if (binding.parsedExpr?.expr) {
    const conditionExpr = convertFromExpr(binding.parsedExpr.expr);
    if (conditionExpr.expiredTime) {
      const expiration = new Date(conditionExpr.expiredTime);
      if (expiration < new Date()) {
        return true;
      }
    }
  }
  return false;
};

export const getUserEmailListInBinding = (binding: Binding): string[] => {
  if (isBindingPolicyExpired(binding)) {
    return [];
  }

  const groupStore = useGroupStore();
  const userStore = useUserStore();
  const emailList = [];

  for (const member of binding.members) {
    if (member.startsWith(groupBindingPrefix)) {
      const group = groupStore.getGroupByIdentifier(member);
      if (!group) {
        continue;
      }

      emailList.push(...group.members.map((m) => extractUserEmail(m.member)));
    } else {
      const email = extractUserEmail(member);
      if (email === ALL_USERS_USER_EMAIL) {
        return userStore.activeUserList.map((user) => user.email);
      } else {
        emailList.push(email);
      }
    }
  }
  return uniq(emailList);
};

export const memberListInIAM = (iamPolicy: IamPolicy) => {
  const userStore = useUserStore();

  const emailList = [];
  // rolesMapByEmail is Map<email, role list>
  const rolesMapByEmail = new Map<string, Set<string>>();
  for (const binding of iamPolicy.bindings) {
    if (isBindingPolicyExpired(binding)) {
      continue;
    }

    const emails = getUserEmailListInBinding(binding);

    for (const email of emails) {
      if (!rolesMapByEmail.has(email)) {
        rolesMapByEmail.set(email, new Set());
      }
      rolesMapByEmail.get(email)?.add(binding.role);
    }
    emailList.push(...emails);
  }

  for (const workspaceLevelProjectMember of userStore.workspaceLevelProjectMembers) {
    emailList.push(workspaceLevelProjectMember.email);
    if (!rolesMapByEmail.has(workspaceLevelProjectMember.email)) {
      rolesMapByEmail.set(workspaceLevelProjectMember.email, new Set());
    }
    for (const role of workspaceLevelProjectMember.roles) {
      if (PRESET_WORKSPACE_ROLES.includes(role)) {
        continue;
      }
      rolesMapByEmail.get(workspaceLevelProjectMember.email)?.add(role);
    }
  }

  const distinctEmailList = uniq(emailList);
  const userList = distinctEmailList.map((email) => {
    const user: ComposedUser = userStore.getUserByEmail(email) ?? unknownUser();
    return user;
  });

  const composedUserList = userList.map((user) => {
    const roleList = rolesMapByEmail.get(user.email) ?? new Set<string>();
    return { user, roleList: [...roleList] };
  });

  return orderBy(
    composedUserList,
    [
      (item) => (item.roleList.includes(PresetRoleType.PROJECT_OWNER) ? 0 : 1),
      (item) => item.user.email,
    ],
    ["asc", "asc"]
  );
};

export const roleListInIAM = (
  iamPolicy: IamPolicy,
  email: string,
  ignoreGroup: boolean = false
) => {
  const groupStore = useGroupStore();
  const userInBinding = getUserEmailInBinding(email);

  const roles = iamPolicy.bindings
    .filter((binding) => {
      if (binding.role === PresetRoleType.WORKSPACE_MEMBER) {
        return false;
      }
      if (isBindingPolicyExpired(binding)) {
        return false;
      }
      for (const member of binding.members) {
        if (member === ALL_USERS_USER_EMAIL) {
          return true;
        }
        if (member === userInBinding) {
          return true;
        }

        if (!ignoreGroup && member.startsWith(groupBindingPrefix)) {
          const group = groupStore.getGroupByIdentifier(member);
          if (!group) {
            continue;
          }

          return group.members.some(
            (m) => m.member === `${userNamePrefix}${email}`
          );
        }
      }
      return false;
    })
    .map((binding) => binding.role);

  if (!roles.some((role) => role.startsWith(`${roleNamePrefix}workspace`))) {
    roles.push(PresetRoleType.WORKSPACE_MEMBER);
  }

  return roles;
};
