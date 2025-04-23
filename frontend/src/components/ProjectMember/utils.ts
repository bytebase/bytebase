import type { Binding } from "@/types/proto/api/v1alpha/iam_policy";
import { displayRoleTitle } from "@/utils";

export const getBindingIdentifier = (binding: Binding): string => {
  const identifier = [displayRoleTitle(binding.role)];
  if (binding.condition && binding.condition.expression) {
    identifier.push(binding.condition.expression);
  }
  return identifier.join(".");
};
