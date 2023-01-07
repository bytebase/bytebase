import { RowStatus } from "./common";
import { MemberId, PrincipalId } from "./id";
import { Principal } from "./principal";

export type MemberStatus = "INVITED" | "ACTIVE";

export type RoleType = "OWNER" | "DBA" | "DEVELOPER";

export type Member = {
  id: MemberId;
  rowStatus: RowStatus;

  // Domain specific fields
  status: MemberStatus;
  role: RoleType;
  principal: Principal;
};

export type MemberCreate = {
  // Domain specific fields
  principalId: PrincipalId;
  status: MemberStatus;
  role: RoleType;
};

export type MemberPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  role?: RoleType;
};
