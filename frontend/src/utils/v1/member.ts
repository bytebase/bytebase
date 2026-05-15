import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import {
  extractGroupEmail,
  serviceAccountToUser,
  useGroupStore,
  useServiceAccountStore,
  useUserStore,
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store";
import {
  extractUserEmail,
  groupNamePrefix,
  serviceAccountNamePrefix,
  workloadIdentityNamePrefix,
} from "@/store/modules/v1/common";
import {
  getGroupEmailInBinding,
  groupBindingPrefix,
  unknownUser,
} from "@/types";
import { PRESET_ROLES } from "@/types/iam/role";
import { GroupSchema } from "@/types/proto-es/v1/group_service_pb";
import type { Binding, IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import { type User, UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type { GroupBinding, MemberBinding } from "@/types/v1/member";
import { convertMemberToFullname, hasWorkspacePermissionV2 } from "@/utils";

const getMemberBinding = (
  member: string,
  searchText: string
): MemberBinding | undefined => {
  const groupStore = useGroupStore();
  const userStore = useUserStore();
  const serviceAccountStore = useServiceAccountStore();
  const workloadIdentityStore = useWorkloadIdentityStore();

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
    let user: User | undefined = unknownUser(member);
    let isPending = false;
    const fullname = convertMemberToFullname(member);
    if (fullname.startsWith(serviceAccountNamePrefix)) {
      const sa = serviceAccountStore.getServiceAccount(fullname);
      user = serviceAccountToUser(sa);
    } else if (fullname.startsWith(workloadIdentityNamePrefix)) {
      const wi = workloadIdentityStore.getWorkloadIdentity(fullname);
      user = workloadIdentityToUser(wi);
    } else {
      const realUser = userStore.getUserByIdentifier(member);
      if (realUser) {
        user = realUser;
      } else {
        const email = extractUserEmail(member);
        user = create(UserSchema, {
          title: email,
          name: fullname,
          email: email,
        });
      }
      // Mark as pending (no principal) only when we can trust the lookup —
      // i.e. the current user has permission to list or get users.
      isPending =
        !realUser &&
        (hasWorkspacePermissionV2("bb.users.list") ||
          hasWorkspacePermissionV2("bb.users.get"));
    }

    memberBinding = {
      type: "users",
      title: user.title,
      user: user,
      binding: member,
      workspaceLevelRoles: new Set<string>(),
      projectRoleBindings: [],
      pending: isPending,
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
          return extractUserEmail(binding.user.name);
        }
        if (binding.group) {
          return extractGroupEmail(binding.group.name);
        }
      },
    ],
    ["asc", "desc"]
  );
};

export interface ProjectRoleBindingGroup {
  role: string;
  bindings: Binding[];
}

export const getProjectRoleBindingKey = (
  binding: Binding,
  index: number
): string => {
  return [
    binding.role,
    binding.condition?.expression ?? "",
    binding.condition?.description ?? "",
    index,
  ].join("::");
};

export const groupProjectRoleBindings = (
  bindings: Binding[]
): ProjectRoleBindingGroup[] => {
  const roleMap = new Map<string, Binding[]>();

  for (const binding of bindings) {
    if (!roleMap.has(binding.role)) {
      roleMap.set(binding.role, []);
    }
    roleMap.get(binding.role)?.push(binding);
  }

  return [...roleMap.keys()]
    .sort((a, b) => {
      const priority = (role: string) => {
        const presetRoleIndex = PRESET_ROLES.indexOf(role);
        if (presetRoleIndex !== -1) {
          return presetRoleIndex;
        }
        return PRESET_ROLES.length;
      };
      return priority(a) - priority(b);
    })
    .map((role) => ({
      role,
      bindings: roleMap.get(role) ?? [],
    }));
};
