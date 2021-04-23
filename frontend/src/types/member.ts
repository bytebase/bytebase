import { MemberId, PrincipalId } from "./id";
import { Principal } from "./principal";

export type RoleType = "OWNER" | "DBA" | "DEVELOPER" | "GUEST";

export type Member = {
  id: MemberId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  role: RoleType;
  principalId: PrincipalId;
};

export type MemberNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  principalId: PrincipalId;
  role: RoleType;
};

export type MemberPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  role: RoleType;
};
