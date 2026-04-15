import { orderBy } from "lodash-es";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";

const getBindingExpirationTimestamp = (
  binding: Binding
): number | undefined => {
  const expression = binding.condition?.expression;
  if (!expression) {
    return undefined;
  }

  const match = expression.match(/request\.time\s*<\s*timestamp\("([^"]+)"\)/);
  if (!match) {
    return undefined;
  }

  const timestamp = Date.parse(match[1]);
  return Number.isNaN(timestamp) ? undefined : timestamp;
};

const isBindingExpired = (binding: Binding): boolean => {
  const expiration = getBindingExpirationTimestamp(binding);
  return expiration !== undefined && expiration < Date.now();
};

export const getUniqueProjectRoleBindings = (
  bindings: Binding[]
): Binding[] => {
  const roleMap = new Map<string, { expired: boolean; binding: Binding }>();

  for (const binding of bindings) {
    const expired = isBindingExpired(binding);
    if (
      !roleMap.has(binding.role) ||
      (roleMap.get(binding.role)?.expired && !expired)
    ) {
      roleMap.set(binding.role, {
        expired,
        binding,
      });
    }
  }

  return orderBy(
    [...roleMap.values()].map((item) => item.binding),
    ["role"]
  );
};
