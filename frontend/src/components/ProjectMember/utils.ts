import { PresetRoleType } from "@/types";
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
  return (
    role !== PresetRoleType.PROJECT_OWNER &&
    checkRoleContainsAnyPermission(
      role,
      "bb.sql.select",
      "bb.sql.ddl",
      "bb.sql.dml",
      "bb.sql.explain",
      "bb.sql.info"
    )
  );
};
