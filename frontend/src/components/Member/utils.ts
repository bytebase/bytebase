import { extractGroupEmail, useGroupStore, useUserStore } from "@/store";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
} from "@/types";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import type { MemberBinding } from "./types";

const getMemberBinding = (
  member: string,
  searchText: string
): MemberBinding | undefined => {
  const groupStore = useGroupStore();
  const userStore = useUserStore();

  if (searchText && !member.toLowerCase().includes(searchText)) {
    return undefined;
  }

  if (member.startsWith(groupBindingPrefix)) {
    const group = groupStore.getGroupByIdentifier(member);
    if (!group) {
      return undefined;
    }
    if (searchText && !group.title.toLowerCase().includes(searchText)) {
      return undefined;
    }
    const email = extractGroupEmail(group.name);

    return {
      type: "groups",
      title: group.title,
      group,
      binding: getGroupEmailInBinding(email),
      workspaceLevelRoles: new Set<string>(),
      projectRoleBindings: [],
    };
  }

  const user = userStore.getUserByIdentifier(member);
  if (!user) {
    return undefined;
  }
  if (searchText && !user.title.toLowerCase().includes(searchText)) {
    return undefined;
  }

  return {
    type: "users",
    title: user.title,
    user,
    binding: getUserEmailInBinding(user.email),
    workspaceLevelRoles: new Set<string>(),
    projectRoleBindings: [],
  };
};

export const getMemberBindingsByRole = ({
  policies,
  searchText,
  ignoreRoles,
}: {
  policies: { level: "WORKSPACE" | "PROJECT"; policy: IamPolicy }[];
  searchText: string;
  ignoreRoles: Set<string>;
}) => {
  // Map<role, Map<member, MemberBinding>>
  const memberMap = new Map<string, MemberBinding>();
  const roleMap = new Map<string, Map<string, MemberBinding>>();
  const search = searchText.trim().toLowerCase();

  const ensureMemberBinding = (member: string) => {
    if (!memberMap.has(member)) {
      const memberBinding = getMemberBinding(member, search);
      if (!memberBinding) {
        return undefined;
      }
      memberMap.set(member, memberBinding);
    }
    return memberMap.get(member);
  };

  for (const policy of policies) {
    for (const binding of policy.policy.bindings) {
      if (ignoreRoles.has(binding.role)) {
        continue;
      }
      if (!roleMap.has(binding.role)) {
        roleMap.set(binding.role, new Map<string, MemberBinding>());
      }
      for (const member of binding.members) {
        const memberBinding = ensureMemberBinding(member);
        if (!memberBinding) {
          continue;
        }
        if (policy.level === "WORKSPACE") {
          memberBinding.workspaceLevelRoles.add(binding.role);
        } else if (policy.level === "PROJECT") {
          memberBinding.projectRoleBindings.push(binding);
        }

        if (!roleMap.get(binding.role)?.has(member)) {
          roleMap.get(binding.role)?.set(member, memberBinding);
        }
      }
    }
  }

  return roleMap;
};

export const getMemberBindings = (
  // Map<role, Map<member, MemberBinding>>
  memberBindingsByRole: Map<string, Map<string, MemberBinding>>
) => {
  const seen = new Set<string>();
  const bindings: MemberBinding[] = [];
  for (const memberBindings of memberBindingsByRole.values()) {
    if (memberBindings.size === 0) {
      continue;
    }
    for (const memberBinding of memberBindings.values()) {
      if (seen.has(memberBinding.binding)) {
        continue;
      }
      seen.add(memberBinding.binding);
      bindings.push(memberBinding);
    }
  }
  return bindings;
};
