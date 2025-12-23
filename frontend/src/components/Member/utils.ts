import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import { extractGroupEmail, useGroupStore, useUserStore } from "@/store";
import {
  extractUserId,
  groupNamePrefix,
  userNamePrefix,
} from "@/store/modules/v1/common";
import {
  getGroupEmailInBinding,
  getUserEmailInBinding,
  groupBindingPrefix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { GroupSchema } from "@/types/proto-es/v1/group_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import { UserSchema, UserType } from "@/types/proto-es/v1/user_service_pb";
import type { GroupBinding, MemberBinding } from "./types";

const getMemberBinding = (
  member: string,
  searchText: string
): MemberBinding | undefined => {
  const groupStore = useGroupStore();
  const userStore = useUserStore();

  let memberBinding: MemberBinding | undefined = undefined;
  if (member.startsWith(groupBindingPrefix)) {
    const g = groupStore.getGroupByIdentifier(member);
    let group: GroupBinding | undefined;
    if (g) {
      group = {
        ...g,
        deleted: false,
      };
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
    let user = userStore.getUserByIdentifier(member);
    if (!user) {
      const email = extractUserId(member);
      user = create(UserSchema, {
        title: email,
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

export const getMemberBindings = ({
  policies,
  searchText,
  ignoreRoles,
}: {
  policies: { level: "WORKSPACE" | "PROJECT"; policy: IamPolicy }[];
  searchText: string;
  ignoreRoles: Set<string>;
}): MemberBinding[] => {
  const memberMap = new Map<string, MemberBinding>();
  const search = searchText.trim().toLowerCase();

  for (const policy of policies) {
    for (const binding of policy.policy.bindings) {
      if (ignoreRoles.has(binding.role)) {
        continue;
      }

      for (const member of binding.members) {
        if (!memberMap.has(member)) {
          const memberBinding = getMemberBinding(member, search);
          if (memberBinding) {
            memberMap.set(member, memberBinding);
          }
        }
        const memberBinding = memberMap.get(member);
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

  return orderBy(
    [...memberMap.values()],
    [
      (binding) => (binding.group ? 0 : 1),
      (binding) => {
        if (binding.user) {
          return extractUserId(binding.user.name);
        }
        if (binding.group) {
          return extractGroupEmail(binding.group.name);
        }
      },
    ],
    ["asc", "desc"]
  );
};
