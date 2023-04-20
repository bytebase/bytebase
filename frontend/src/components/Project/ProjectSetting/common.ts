import type { Member, Principal, ProjectRoleType } from "@/types";

export type ComposedPrincipal = {
  email: string;
  member: Member;
  principal: Principal;
  roleList: ProjectRoleType[];
};
