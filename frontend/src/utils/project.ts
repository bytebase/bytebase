import {
  DEFAULT_PROJECT_ID,
  Principal,
  PrincipalId,
  type Project,
} from "../types";

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

export const isMemberOfProject = (
  project: Project,
  userOrId: Principal | PrincipalId
) => {
  const id = typeof userOrId === "object" ? userOrId.id : userOrId;
  return (
    project.memberList.findIndex((member) => member.principal.id === id) >= 0
  );
};
