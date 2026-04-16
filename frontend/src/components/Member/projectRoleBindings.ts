import { PRESET_ROLES } from "@/types/iam/role";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";

export interface ProjectRoleBindingGroup {
  role: string;
  bindings: Binding[];
}

export const getProjectRoleBindingKey = (
  binding: Binding,
  index: number
): string => {
  return [
    binding.role,
    binding.condition?.expression ?? "",
    binding.condition?.description ?? "",
    index,
  ].join("::");
};

export const groupProjectRoleBindings = (
  bindings: Binding[]
): ProjectRoleBindingGroup[] => {
  const roleMap = new Map<string, Binding[]>();

  for (const binding of bindings) {
    if (!roleMap.has(binding.role)) {
      roleMap.set(binding.role, []);
    }
    roleMap.get(binding.role)?.push(binding);
  }

  return [...roleMap.keys()]
    .sort((a, b) => {
      const priority = (role: string) => {
        const presetRoleIndex = PRESET_ROLES.indexOf(role);
        if (presetRoleIndex !== -1) {
          return presetRoleIndex;
        }
        return PRESET_ROLES.length;
      };
      return priority(a) - priority(b);
    })
    .map((role) => ({
      role,
      bindings: roleMap.get(role) ?? [],
    }));
};
