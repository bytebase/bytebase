import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { displayRoleTitle } from "@/utils";

export const getBindingIdentifier = (binding: Binding): string => {
  const identifier = [displayRoleTitle(binding.role)];
  if (binding.condition && binding.condition.expression) {
    identifier.push(binding.condition.expression);
  }
  return identifier.join(".");
};
