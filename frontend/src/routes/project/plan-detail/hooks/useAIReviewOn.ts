import { useEffect } from "react";
import { projectAIReviewSwitch } from "@/components/sql-review/ReviewAIPolicyField";
import { useAppStore } from "@/stores/app";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PROJECT_NAME_PREFIX } from "../shell/constants";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { aiReviewEnabled } from "../utils/reviewRun";

// Whether the page's project gets the AI review. Until the project's policy
// is read, or for a reader who cannot read it, the project follows the
// workspace.
export function useAIReviewOn(): boolean {
  const page = usePlanDetailContext();
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
  return aiReviewEnabled(serverInfo, projectSwitch);
}
