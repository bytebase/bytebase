import { uniq } from "lodash-es";
import {
  ensureGroupIdentifier,
  extractServiceAccountId,
  extractWorkloadIdentityId,
  groupNamePrefix,
  useGroupStore,
  useWorkspaceV1Store,
} from "@/store";
import {
  serviceAccountNamePrefix,
  userNamePrefix,
  workloadIdentityNamePrefix,
} from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  groupBindingPrefix,
  serviceAccountBindingPrefix,
  workloadIdentityBindingPrefix,
} from "@/types";
import type { Binding, IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import { ensureUserFullName } from "@/utils";
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
}: {
  binding: Binding;
  ignoreGroup: boolean;
}): string[] => {
  if (isBindingPolicyExpired(binding)) {
    return [];
  }

  const groupStore = useGroupStore();
  const fullnameList = [];

  for (const member of binding.members) {
    const fullname = convertMemberToFullname(member);
    if (fullname.startsWith(groupNamePrefix)) {
      if (ignoreGroup) {
        continue;
      }
      const group = groupStore.getGroupByIdentifier(member);
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
  targetRole?: string
): Map<string, Set<string>> => {
  const workspaceStore = useWorkspaceV1Store();
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

    const fullnames = getUserListInBinding({ binding, ignoreGroup: false });
    for (const fullname of fullnames) {
      if (!rolesMapByName.has(fullname)) {
        rolesMapByName.set(fullname, new Set());
      }
      rolesMapByName.get(fullname)?.add(binding.role);
    }
  }

  // Handle workspace level project roles.
  for (const [role, userSet] of workspaceStore.roleMapToUsers.entries()) {
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
