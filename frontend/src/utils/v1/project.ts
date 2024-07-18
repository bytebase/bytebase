import { usePermissionStore } from "@/store/modules/v1/permission";
import {
  DEFAULT_PROJECT_NAME,
  PresetRoleType,
  type ComposedProject,
  type ComposedUser,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import type { Project } from "@/types/proto/v1/project_service";

export const extractProjectResourceName = (name: string) => {
  const pattern = /(?:^|\/)projects\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isMemberOfProjectV1 = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return usePermissionStore().roleListInProjectV1(project, user).length > 0;
};

export const isOwnerOfProjectV1 = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return usePermissionStore()
    .roleListInProjectV1(project, user)
    .includes(PresetRoleType.PROJECT_OWNER);
};

export const isDeveloperOfProjectV1 = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return usePermissionStore()
    .roleListInProjectV1(project, user)
    .includes(PresetRoleType.PROJECT_DEVELOPER);
};

export const isViewerOfProjectV1 = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return usePermissionStore()
    .roleListInProjectV1(project, user)
    .includes(PresetRoleType.PROJECT_VIEWER);
};

export function projectV1Name(project: Project) {
  if (project.name === DEFAULT_PROJECT_NAME) {
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
