import { RowStatus } from "./common";
import { MemberID, PrincipalID } from "./id";
import { Principal } from "./principal";

export type MemberStatus = "INVITED" | "ACTIVE";

export type RoleType = "OWNER" | "DBA" | "DEVELOPER";

export type Member = {
  id: MemberID;

  // Standard fields
  rowStatus: RowStatus;
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  status: MemberStatus;
  role: RoleType;
  principal: Principal;
};

export type MemberCreate = {
  // Domain specific fields
  principalID: PrincipalID;
  status: MemberStatus;
  role: RoleType;
};

export type MemberPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  role?: RoleType;
};
