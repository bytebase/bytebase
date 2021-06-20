import { MemberId, PrincipalId } from "./id";
import { Principal } from "./principal";

export type RoleType = "OWNER" | "DBA" | "DEVELOPER";

export type Member = {
  id: MemberId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  role: RoleType;
  principal: Principal;
};

export type MemberCreate = {
  // Domain specific fields
  principalId: PrincipalId;
  role: RoleType;
};

export type MemberPatch = {
  // Domain specific fields
  role: RoleType;
};
