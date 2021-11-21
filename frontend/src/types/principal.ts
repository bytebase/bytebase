// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,

import { PrincipalID } from "./id";
import { RoleType } from "./member";

// we may support application/bot identity.
export type PrincipalType = "END_USER" | "SYSTEM_BOT";

export type Principal = {
  id: PrincipalID;

  // Standard fields
  // Unlike other models, we use creatorID and updaterID instead of converting
  // them to full principal object to avoid recursive issue.
  creatorID: PrincipalID;
  createdTs: number;
  updaterID: PrincipalID;
  updatedTs: number;

  // Domain specific fields
  type: PrincipalType;
  name: string;
  email: string;
  role: RoleType;
};

export type PrincipalCreate = {
  // Domain specific fields
  name: string;
  email: string;
};

export type PrincipalPatch = {
  // Domain specific fields
  name?: string;
  password?: string;
};
