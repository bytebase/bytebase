import { ProjectRoleType } from "@/types";
import type { IamPolicy } from "@/types/proto/v1/project_service";

const tidyUpPolicy = (policy: IamPolicy) => {
  policy.bindings = policy.bindings.filter(
    (binding) => binding.members.length > 0
  );
};

export const removeRoleFromProjectIamPolicy = (
  policy: IamPolicy,
  user: string,
  role: ProjectRoleType
) => {
  const binding = policy.bindings.find((binding) => binding.role === role);
  if (binding) {
    const index = binding.members.indexOf(user);
    if (index >= 0) {
      binding.members.splice(index, 1);

      tidyUpPolicy(policy);
    }
  }
};

export const removeUserFromProjectIamPolicy = (
  policy: IamPolicy,
  user: string
) => {
  policy.bindings.forEach((binding) => {
    const index = binding.members.indexOf(user);
    if (index >= 0) {
      binding.members.splice(index, 1);
    }
  });
  tidyUpPolicy(policy);
};

export const addRoleToProjectIamPolicy = (
  policy: IamPolicy,
  user: string,
  role: ProjectRoleType
) => {
  policy.bindings.push({
    role,
    members: [user],
  });
};
