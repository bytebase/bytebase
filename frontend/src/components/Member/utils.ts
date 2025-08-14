import { create } from "@bufbuild/protobuf";
import { extractGroupEmail, useGroupStore, useUserStore } from "@/store";
import {
  extractUserId,
  userNamePrefix,
  groupNamePrefix,
} from "@/store/modules/v1/common";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { GroupSchema } from "@/types/proto-es/v1/group_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import type { MemberBinding, GroupBinding } from "./types";

const getMemberBinding = async (
  member: string,
  searchText: string
): Promise<MemberBinding | undefined> => {
  const groupStore = useGroupStore();
  const userStore = useUserStore();

  let memberBinding: MemberBinding | undefined = undefined;
  if (member.startsWith(groupBindingPrefix)) {
    let group: GroupBinding | undefined;
    try {
      const g = await groupStore.getOrFetchGroupByIdentifier(member);
      if (g) {
        group = {
          ...g,
          deleted: false,
        };
      }
    } catch {
      // nothing
    }
    if (!group) {
      const email = extractGroupEmail(member);
      group = {
        ...create(GroupSchema, {
          name: `${groupNamePrefix}${email}`,
          title: email,
        }),
        deleted: true,
      };
    }

    memberBinding = {
      type: "groups",
      title: group.title,
      group,
      binding: getGroupEmailInBinding(extractGroupEmail(group.name)),
      workspaceLevelRoles: new Set<string>(),
      projectRoleBindings: [],
    };
  } else {
    let user: User | undefined;
    try {
      user = await userStore.getOrFetchUserByIdentifier(member);
    } catch {
      // nothing
    }

    if (!user) {
      const email = extractUserId(member);
      user = create(UserSchema, {
        title: member,
        name: `${userNamePrefix}${email}`,
        email: email,
        userType: UserType.USER,
        state: State.DELETED,
      });
    }
    memberBinding = {
      type: "users",
      title: user.title,
      user: user,
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

export const getMemberBindings = async ({
  policies,
  searchText,
  ignoreRoles,
}: {
  policies: { level: "WORKSPACE" | "PROJECT"; policy: IamPolicy }[];
  searchText: string;
  ignoreRoles: Set<string>;
}): Promise<MemberBinding[]> => {
  const memberMap = new Map<string, MemberBinding>();
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
      }
    }
  }

  return [...memberMap.values()];
};
