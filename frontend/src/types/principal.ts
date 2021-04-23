// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,

import { PrincipalId } from "./id";
import { RoleType } from "./member";

// we may support application/bot identity.
export type PrincipalStatus = "UNKNOWN" | "INVITED" | "ACTIVE";

export type PrincipalType = "END_USER" | "BOT";

export type Principal = {
  id: PrincipalId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  status: PrincipalStatus;
  type: PrincipalType;
  name: string;
  email: string;
  role: RoleType;
};

export type PrincipalNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  email: string;
};

export type PrincipalPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  name?: string;
};
