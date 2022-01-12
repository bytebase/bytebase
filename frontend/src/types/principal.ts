// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,

import { AuthProviderType } from ".";
import { PrincipalId } from "./id";
import { RoleType } from "./member";

// we may support application/bot identity.
export type PrincipalType = "END_USER" | "SYSTEM_BOT";

export type Principal = {
  id: PrincipalId;

  // Standard fields
  // Unlike other models, we use creatorId and updaterId instead of converting
  // them to full principal object to avoid recursive issue.
  creatorId: PrincipalId;
  createdTs: number;
  updaterId: PrincipalId;
  updatedTs: number;

  // Domain specific fields
  authProvider: AuthProviderType;
  type: PrincipalType;
  name: string;
  email: string;
  role: RoleType;
};

export type PrincipalCreate = {
  // Domain specific fields
  authProvider: AuthProviderType;
  name: string;
  email: string;
};

export type PrincipalPatch = {
  // Domain specific fields
  name?: string;
  password?: string;
};
