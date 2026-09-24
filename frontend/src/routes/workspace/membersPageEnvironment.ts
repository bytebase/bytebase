import {
  type DirectExecutionScope,
  directExecutionScopeFromCondition,
} from "@/components/role-grant/directExecutionScope";
import { getRoleEnvironmentLimitationKind } from "@/lib/project-member/utils";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { convertFromExpr } from "@/utils/issue/cel";

// undefined ⇔ the role carries no DDL/DML permission ⇔ nothing to show.
export const getProjectRoleBindingDirectExecutionScope = (
  binding: Binding
): DirectExecutionScope | undefined => {
  if (getRoleEnvironmentLimitationKind(binding.role) === undefined) {
    return undefined;
  }
  const condition = binding.parsedExpr
    ? convertFromExpr(binding.parsedExpr)
    : undefined;
  return directExecutionScopeFromCondition(
    condition,
    binding.condition?.expression ?? ""
  );
};
