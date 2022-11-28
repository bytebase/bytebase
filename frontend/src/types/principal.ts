// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,
import { PrincipalId } from "./id";
import { RoleType } from "./member";

// we may support application/bot identity.
export type PrincipalType = "END_USER" | "SYSTEM_BOT" | "SERVICE_ACCOUNT";

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
  type: PrincipalType;
  name: string;
  email: string;
  role: RoleType;
  token: string;
};

export type PrincipalCreate = {
  // Domain specific fields
  name: string;
  email: string;
  type: PrincipalType;
};

export type PrincipalPatch = {
  // Domain specific fields
  name?: string;
  password?: string;
  email?: string;
  type: PrincipalType;
};
