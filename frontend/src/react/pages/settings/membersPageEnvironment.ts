import { getRoleEnvironmentLimitationKind } from "@/react/lib/project-member/utils";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { convertFromExpr } from "@/utils/issue/cel";

export type ProjectRoleBindingEnvironmentLimitationState =
  | {
      type: "unrestricted";
    }
  | {
      environments: string[];
      type: "restricted";
    };

export const getProjectRoleBindingEnvironmentLimitationState = (
  binding: Binding
): ProjectRoleBindingEnvironmentLimitationState | undefined => {
  if (getRoleEnvironmentLimitationKind(binding.role) === undefined) {
    return undefined;
  }

  if (!binding.parsedExpr) {
    return {
      type: "unrestricted",
    };
  }

  const environments = convertFromExpr(binding.parsedExpr).environments;
  if (environments === undefined) {
    return {
      type: "unrestricted",
    };
  }

  return {
    environments,
    type: "restricted",
  };
};
