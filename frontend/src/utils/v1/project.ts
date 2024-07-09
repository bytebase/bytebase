import { orderBy, uniq } from "lodash-es";
import { extractUserEmail, useUserStore, useUserGroupStore } from "@/store";
import { usePermissionStore } from "@/store/modules/v1/permission";
import {
  ALL_USERS_USER_EMAIL,
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_ID,
  PresetRoleType,
  groupBindingPrefix,
  PRESET_WORKSPACE_ROLES,
  type ComposedProject,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import type { IamPolicy, Binding } from "@/types/proto/v1/iam_policy";
import type { Project } from "@/types/proto/v1/project_service";
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

export const extractProjectResourceName = (name: string) => {
  const pattern = /(?:^|\/)projects\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isMemberOfProjectV1 = (project: ComposedProject, user: User) => {
  return usePermissionStore().roleListInProjectV1(project, user).length > 0;
};

export const isOwnerOfProjectV1 = (project: ComposedProject, user: User) => {
  return usePermissionStore()
    .roleListInProjectV1(project, user)
    .includes(PresetRoleType.PROJECT_OWNER);
};

export const isDeveloperOfProjectV1 = (
  project: ComposedProject,
  user: User
) => {
  return usePermissionStore()
    .roleListInProjectV1(project, user)
    .includes(PresetRoleType.PROJECT_DEVELOPER);
};

export const isViewerOfProjectV1 = (project: ComposedProject, user: User) => {
  return usePermissionStore()
    .roleListInProjectV1(project, user)
    .includes(PresetRoleType.PROJECT_VIEWER);
};

export const getUserEmailListInBinding = (binding: Binding): string[] => {
  if (isBindingPolicyExpired(binding)) {
    return [];
  }

  const groupStore = useUserGroupStore();
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

export const memberListInProjectV1 = (iamPolicy: IamPolicy) => {
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
    const user =
      userStore.getUserByEmail(email) ??
      User.fromJSON({
        name: `users/${UNKNOWN_ID}`,
        email,
        title: "<<Unknown user>>",
      });
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

export function projectV1Name(project: Project) {
  if (project.name === DEFAULT_PROJECT_V1_NAME) {
    return "Unassigned";
  }

  const parts = [project.title];
  if (project.state === State.DELETED) {
    parts.push("(Archived)");
  }
  return parts.join(" ");
}

export function filterProjectV1ListByKeyword<T extends Project>(
  projectList: T[],
  keyword: string
) {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) return projectList;
  return projectList.filter((project) => {
    return (
      project.title.toLowerCase().includes(keyword) ||
      project.key.toLowerCase().includes(keyword)
    );
  });
}
