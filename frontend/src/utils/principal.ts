import { Principal, Project } from "@/types";

export const isOwnerOfProject = (
  principal: Principal,
  project: Project
): boolean => {
  return (
    project.memberList.findIndex(
      (member) =>
        member.roleProvider === project.roleProvider &&
        member.role === "OWNER" &&
        member.principal.id === principal.id
    ) >= 0
  );
};
