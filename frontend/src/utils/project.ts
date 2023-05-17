import {
  DEFAULT_PROJECT_ID,
  Principal,
  PrincipalId,
  ProjectMember,
  ProjectRoleType,
  type Project,
} from "../types";
import { hasProjectPermission, ProjectPermissionType } from "./role";

export function projectName(project: Project) {
  if (project.id === DEFAULT_PROJECT_ID) {
    return "Unassigned";
  }

  let name = project.name;
  if (project.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}

/**
 * Returns the members for a user in a project
 * @param project
 * @param user
 * @param roles roles filter. Empty array `[]` to "ALL"
 */
export const memberListInProject = (
  project: Project,
  user: Principal,
  roles: ProjectRoleType[] = []
): ProjectMember[] => {
  const memberList = project.memberList.filter(
    (member) => member.principal.id === user.id
  );
  if (roles.length === 0) return memberList;

  return memberList.filter((member) => roles.includes(member.role));
};

/**
 * Returns one of the membership in a project
 * @param project
 * @param user
 * @param role Empty to "ANY"
 */
export const memberInProject = (
  project: Project,
  user: Principal,
  role?: ProjectRoleType
): ProjectMember | undefined => {
  if (role) {
    return project.memberList.find((member) => member.principal.id === user.id);
  }
  return project.memberList.find(
    (member) => member.principal.id === user.id && member.role === role
  );
};

export const hasPermissionInProject = (
  project: Project,
  user: Principal,
  permission: ProjectPermissionType
) => {
  const memberList = memberListInProject(project, user);
  if (
    memberList.some((member) => hasProjectPermission(permission, member.role))
  ) {
    return true;
  }

  return false;
};

export const isMemberOfProject = (
  project: Project,
  userOrId: Principal | PrincipalId
) => {
  const id = typeof userOrId === "object" ? userOrId.id : userOrId;
  return (
    project.memberList.findIndex((member) => member.principal.id === id) >= 0
  );
};
