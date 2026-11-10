import { useEffect } from "react";
import {
  effectiveWorkspaceRules,
  reviewRulesOfPolicy,
} from "@/components/sql-review/standardRules";
import { useWorkspaceResourceName } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import type { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export type EffectiveStandardRules = {
  rules: ReviewRuleType[];
  source: "project" | "workspace";
};

// The standard rules the rule review applies to a project: its own enforced
// policy, else the workspace's. Undefined until known, and when the reader
// cannot see the workspace rules a project without its own follows.
export function useEffectiveStandardRules(
  projectName: string
): EffectiveStandardRules | undefined {
  const workspace = useWorkspaceResourceName();
  const canReadWorkspace = hasWorkspacePermissionV2("bb.policies.get");
  const projectPolicy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath: projectName,
      policyType: PolicyType.REVIEW_RULE,
    })
  );
  const workspacePolicy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath: workspace,
      policyType: PolicyType.REVIEW_RULE,
    })
  );
  useEffect(() => {
    const store = useAppStore.getState();
    void store
      .fetchPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.REVIEW_RULE,
      })
      .catch(() => undefined);
    if (workspace && canReadWorkspace) {
      void store
        .fetchPolicyByParentAndType({
          parentPath: workspace,
          policyType: PolicyType.REVIEW_RULE,
        })
        .catch(() => undefined);
    }
  }, [canReadWorkspace, projectName, workspace]);

  const projectRules = reviewRulesOfPolicy(projectPolicy);
  if (projectRules) return { rules: projectRules, source: "project" };
  const workspaceRules = canReadWorkspace
    ? effectiveWorkspaceRules(workspacePolicy)
    : undefined;
  return workspaceRules
    ? { rules: workspaceRules, source: "workspace" }
    : undefined;
}
