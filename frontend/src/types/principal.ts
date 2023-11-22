// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,
import { PrincipalId } from "./id";
import { RoleType } from "./member";

// we may support application/bot identity.
export type PrincipalType = "END_USER" | "SYSTEM_BOT" | "SERVICE_ACCOUNT";

export type Principal = {
  id: PrincipalId;

  // Domain specific fields
  type: PrincipalType;
  name: string;
  email: string;
  role: RoleType;
  serviceKey: string;
};
