import { orderBy, uniq } from "lodash-es";
import {
  extractUserEmail,
  useUserStore,
  useGroupStore,
  useWorkspaceV1Store,
} from "@/store";
import { roleNamePrefix } from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  PresetRoleType,
  groupBindingPrefix,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
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
  const userStore = useUserStore();
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

export const memberListInProjectIAM = (
  iamPolicy: IamPolicy,
  targetRole?: string
) => {
  const userStore = useUserStore();
  const workspaceStore = useWorkspaceV1Store();

  const emailSet = new Set<string>();
  // rolesMapByEmail is Map<email, role list>
  const rolesMapByEmail = new Map<string, Set<string>>();

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
      if (!rolesMapByEmail.has(email)) {
        rolesMapByEmail.set(email, new Set());
      }
      rolesMapByEmail.get(email)?.add(binding.role);
      emailSet.add(email);
    }
  }

  // Handle workspace level project roles.
  for (const [email, roleSet] of workspaceStore.emailMapToRoles.entries()) {
    emailSet.add(email);
    if (!rolesMapByEmail.has(email)) {
      rolesMapByEmail.set(email, new Set());
    }
    for (const role of roleSet.values()) {
      if (PRESET_WORKSPACE_ROLES.includes(role)) {
        continue;
      }
      if (targetRole && role !== targetRole) {
        continue;
      }
      rolesMapByEmail.get(email)?.add(role);
    }
  }

  const composedUserList: {
    user: User;
    roleList: string[];
  }[] = [];
  for (const email of emailSet.values()) {
    const user = userStore.getUserByEmail(email);
    if (!user) {
      continue;
    }
    const roleList = rolesMapByEmail.get(email);
    if (!roleList || roleList.size === 0) {
      continue;
    }
    composedUserList.push({
      user,
      roleList: [...roleList],
    });
  }

  return orderBy(
    composedUserList,
    [
      (item) => (item.roleList.includes(PresetRoleType.PROJECT_OWNER) ? 0 : 1),
      (item) => item.user.email,
    ],
    ["asc", "asc"]
  );
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
