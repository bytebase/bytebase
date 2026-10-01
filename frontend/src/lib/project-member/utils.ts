import { displayRoleTitleFromList } from "@/lib/role";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types/iam";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { checkRoleContainsAnyPermission } from "@/utils";

export const getBindingIdentifier = (binding: Binding): string => {
  const identifier = [
    displayRoleTitleFromList(binding.role, useAppStore.getState().roleList),
  ];
  if (binding.condition && binding.condition.expression) {
    identifier.push(binding.condition.expression);
  }
  return identifier.join(".");
};

export const roleHasDatabaseLimitation = (role: string) => {
  return checkRoleContainsAnyPermission(
    role,
    "bb.sql.select",
    "bb.sql.ddl",
    "bb.sql.dml",
    "bb.sql.explain",
    "bb.sql.info"
  );
};

// {{kind}} is spliced raw into translated strings — do not localize.
export type EnvLimitationKind = "DDL" | "DML" | "DDL/DML";

// undefined ⇔ the role carries no DDL/DML permission ⇔ no direct-execution
// field, callout, or row to show.
export const getRoleEnvironmentLimitationKind = (
  role: string
): EnvLimitationKind | undefined => getRolesEnvironmentLimitationKind([role]);

// Roles granted together carry the union of their permissions; unknown roles
// contribute nothing.
export const getRolesEnvironmentLimitationKind = (
  roles: string[]
): EnvLimitationKind | undefined => {
  const has = (permission: Permission) =>
    roles.some((role) => checkRoleContainsAnyPermission(role, permission));
  const hasDDL = has("bb.sql.ddl");
  const hasDML = has("bb.sql.dml");
  if (hasDDL && hasDML) return "DDL/DML";
  if (hasDDL) return "DDL";
  if (hasDML) return "DML";
  return undefined;
};
