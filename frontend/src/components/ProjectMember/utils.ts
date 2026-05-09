import { useRoleStore } from "@/store";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { checkRoleContainsAnyPermission, displayRoleTitle } from "@/utils";

export const getBindingIdentifier = (binding: Binding): string => {
  const identifier = [displayRoleTitle(binding.role)];
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

// undefined ⇔ role has no env-scoped permissions ⇔ caller hides the env section.
// Reads the role once (vs. two checkRoleContainsAnyPermission calls) so the
// hot path on member-list / drawer renders touches the role store once.
export const getRoleEnvironmentLimitationKind = (
  role: string
): EnvLimitationKind | undefined => {
  const r = useRoleStore().getRoleByName(role);
  if (!r) return undefined;
  const perms = new Set(r.permissions);
  const hasDDL = perms.has("bb.sql.ddl");
  const hasDML = perms.has("bb.sql.dml");
  if (hasDDL && hasDML) return "DDL/DML";
  if (hasDDL) return "DDL";
  if (hasDML) return "DML";
  return undefined;
};
