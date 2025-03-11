import { extractGroupEmail, useGroupStore, useUserStore } from "@/store";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import type { MemberBinding } from "./types";

const getMemberBinding = async (
  member: string,
  searchText: string
): Promise<MemberBinding | undefined> => {
  const groupStore = useGroupStore();
  const userStore = useUserStore();

  let memberBinding: MemberBinding | undefined = undefined;
  if (member.startsWith(groupBindingPrefix)) {
    const group = await groupStore.getOrFetchGroupByIdentifier(member);
    if (!group) {
      return undefined;
    }
    const email = extractGroupEmail(group.name);

    memberBinding = {
      type: "groups",
      title: group.title,
      group,
      binding: getGroupEmailInBinding(email),
      workspaceLevelRoles: new Set<string>(),
      projectRoleBindings: [],
    };
  } else {
    const user = await userStore.getOrFetchUserByIdentifier(member);
    if (!user) {
      return undefined;
    }
    if (user.state !== State.ACTIVE) {
      return undefined;
    }
    memberBinding = {
      type: "users",
      title: user.title,
      user,
      binding: getUserEmailInBinding(user.email),
      workspaceLevelRoles: new Set<string>(),
      projectRoleBindings: [],
    };
  }

  if (searchText && memberBinding) {
    if (
      !memberBinding.binding.toLowerCase().includes(searchText) &&
      !memberBinding.title.toLowerCase().includes(searchText)
    ) {
      return undefined;
    }
  }

  return memberBinding;
};

// getMemberBindingsByRole returns a map from the input policies.
// The map will in the Map<roles/{role}, Map<user:{email} or group:{email}, MemberBinding>> format
export const getMemberBindingsByRole = async ({
  policies,
  searchText,
  ignoreRoles,
}: {
  policies: { level: "WORKSPACE" | "PROJECT"; policy: IamPolicy }[];
  searchText: string;
  ignoreRoles: Set<string>;
}): Promise<Map<string, Map<string, MemberBinding>>> => {
  // Map<role, Map<member, MemberBinding>>
  const memberMap = new Map<string, MemberBinding>();
  const roleMap = new Map<string, Map<string, MemberBinding>>();
  const search = searchText.trim().toLowerCase();

  const ensureMemberBinding = async (member: string) => {
    if (!memberMap.has(member)) {
      try {
        const memberBinding = await getMemberBinding(member, search);
        if (!memberBinding) {
          return undefined;
        }
        memberMap.set(member, memberBinding);
      } catch {}
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
        const memberBinding = await ensureMemberBinding(member);
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
