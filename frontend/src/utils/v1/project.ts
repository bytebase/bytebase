import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { IamPolicy, Project } from "@/types/proto/v1/project_service";
import {
  extractRoleResourceName,
  hasProjectPermission,
  ProjectPermissionType,
} from "../role";

export const extractProjectResourceName = (name: string) => {
  const pattern = /(?:^|\/)projects\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const hasPermissionInProjectV1 = (
  policy: IamPolicy,
  user: User,
  permission: ProjectPermissionType
) => {
  return policy.bindings.some((binding) => {
    if (binding.members.includes(`user:${user.email}`)) {
      return hasProjectPermission(
        permission,
        extractRoleResourceName(binding.role)
      );
    }
    return false;
  });
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
