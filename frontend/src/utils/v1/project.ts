import { orderBy, uniq } from "lodash-es";
import {
  extractUserEmail,
  useUserStore,
  useUserGroupStore,
  extractGroupEmail,
} from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_ID,
  PresetRoleType,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import type { IamPolicy } from "@/types/proto/v1/iam_policy";
import type { Project } from "@/types/proto/v1/project_service";

export const extractProjectResourceName = (name: string) => {
  const pattern = /(?:^|\/)projects\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const roleListInProjectV1 = (iamPolicy: IamPolicy, user: User) => {
  const groupStore = useUserGroupStore();

  return iamPolicy.bindings
    .filter((binding) => {
      for (const member of binding.members) {
        if (member === ALL_USERS_USER_EMAIL) {
          return true;
        }
        if (member === `user:${user.email}`) {
          return true;
        }

        if (member.startsWith("group:")) {
          const groupEmail = extractGroupEmail(member);
          const group = groupStore.getGroupByEmail(groupEmail);
          if (!group) {
            continue;
          }

          return group.members.some(
            (m) => m.member === `${userNamePrefix}${user.email}`
          );
        }

        return false;
      }
    })
    .map((binding) => binding.role);
};

export const isMemberOfProjectV1 = (iamPolicy: IamPolicy, user: User) => {
  return roleListInProjectV1(iamPolicy, user).length > 0;
};

export const isOwnerOfProjectV1 = (iamPolicy: IamPolicy, user: User) => {
  return roleListInProjectV1(iamPolicy, user).includes(
    PresetRoleType.PROJECT_OWNER
  );
};

export const isDeveloperOfProjectV1 = (iamPolicy: IamPolicy, user: User) => {
  return roleListInProjectV1(iamPolicy, user).includes(
    PresetRoleType.PROJECT_DEVELOPER
  );
};

export const isViewerOfProjectV1 = (iamPolicy: IamPolicy, user: User) => {
  return roleListInProjectV1(iamPolicy, user).includes(
    PresetRoleType.PROJECT_VIEWER
  );
};

export const memberListInProjectV1 = (iamPolicy: IamPolicy) => {
  const userStore = useUserStore();
  const groupStore = useUserGroupStore();

  const emailList = [];
  for (const binding of iamPolicy.bindings) {
    for (const member of binding.members) {
      if (member.startsWith("group:")) {
        const groupEmail = extractGroupEmail(member);
        const group = groupStore.getGroupByEmail(groupEmail);
        if (!group) {
          continue;
        }

        emailList.push(...group.members.map((m) => extractUserEmail(m.member)));
      } else {
        emailList.push(extractUserEmail(member));
      }
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
  const usersByRole = iamPolicy.bindings.map((binding) => {
    return {
      role: binding.role,
      emailList: new Set(binding.members.map(extractUserEmail)),
    };
  });
  const composedUserList = userList.map((user) => {
    const roleList = usersByRole
      .filter((binding) => binding.emailList.has(user.email))
      .map((binding) => binding.role);
    return { user, roleList };
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
