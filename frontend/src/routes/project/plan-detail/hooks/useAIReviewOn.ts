import { useEffect } from "react";
import { projectAIReviewSwitch } from "@/components/sql-review/ReviewAIPolicyField";
import { usePlanFeature } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { PROJECT_NAME_PREFIX } from "../shell/constants";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { aiReviewEnabled } from "../utils/reviewRun";

// Whether the page's project gets the AI review: the workspace's plan includes
// it, and its switch is on. Until the project's policy is read, or for a reader
// who cannot read it, the project follows the workspace's switch.
export function useAIReviewOn(): boolean {
  const page = usePlanDetailContext();
  const licensed = usePlanFeature(PlanFeature.FEATURE_AI_SQL_REVIEW);
  const projectName = `${PROJECT_NAME_PREFIX}${page.projectId}`;
  const serverInfo = useAppStore((state) => state.serverInfo);
  const aiEnabled = serverInfo?.aiEnabled === true;
  const projectSwitch = useAppStore((state) =>
    projectAIReviewSwitch(
      state.getPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.REVIEW_AI,
      })
    )
  );
  useEffect(() => {
    if (!aiEnabled) return;
    void useAppStore
      .getState()
      .getOrFetchPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.REVIEW_AI,
      })
      .catch(() => undefined);
  }, [aiEnabled, projectName]);
  return licensed && aiReviewEnabled(serverInfo, projectSwitch);
}
