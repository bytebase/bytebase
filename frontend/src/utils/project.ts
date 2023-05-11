import { orderBy, uniq } from "lodash-es";
import {
  extractUserEmail,
  useMemberStore,
  useProjectIamPolicyStore,
  useProjectStore,
  useUserStore,
} from "@/store";
import {
  DEFAULT_PROJECT_ID,
  Principal,
  PrincipalId,
  ProjectMember,
  ProjectRoleType,
  ProjectRoleTypeOwner,
  type Project,
  ProjectId,
  ComposedPrincipal,
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

export const isOwnerOfProject = (
  project: Project,
  userOrId: Principal | PrincipalId
): boolean => {
  const id = typeof userOrId === "object" ? userOrId.id : userOrId;
  return (
    project.memberList.findIndex(
      (member) =>
        member.role === ProjectRoleTypeOwner && member.principal.id === id
    ) >= 0
  );
};

export const getProjectMemberList = async (projectId: ProjectId) => {
  const projectStore = useProjectStore();
  const userStore = useUserStore();
  const memberStore = useMemberStore();

  const project = projectStore.getProjectById(projectId);
  const iamPolicy = await useProjectIamPolicyStore().getOrFetchProjectIamPolicy(
    `projects/${project.resourceId}`
  );
  const distinctUserResourceNameList = uniq(
    iamPolicy.bindings.flatMap((binding) => binding.members)
  );
  const userEmailList = distinctUserResourceNameList.map((user) =>
    extractUserEmail(user)
  );
  const composedUserList = userEmailList.map((email) => {
    const user = userStore.getUserByEmail(email);
    const member = memberStore.memberByEmail(email);
    return { email, user, member };
  });
  const usersByRole = iamPolicy.bindings.map((binding) => {
    return {
      role: binding.role,
      users: new Set(binding.members),
    };
  });
  const composedPrincipalList = composedUserList.map<ComposedPrincipal>(
    ({ email, member }) => {
      const resourceName = `user:${email}`;
      const roleList = usersByRole
        .filter((binding) => binding.users.has(resourceName))
        .map((binding) => binding.role);
      return {
        email,
        member,
        principal: member.principal,
        roleList,
      };
    }
  );

  return orderBy(
    composedPrincipalList,
    [
      (item) => (item.roleList.includes("roles/OWNER") ? 0 : 1),
      (item) => item.principal.id,
    ],
    ["asc", "asc"]
  );
};
