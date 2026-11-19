import { usePlanFeature } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { aiReviewEnabled } from "../utils/reviewRun";

// Whether the plan gets the AI review: the workspace's plan includes it, and
// the workspace has AI and its review switch on.
export function useAIReviewOn(): boolean {
  const licensed = usePlanFeature(PlanFeature.FEATURE_AI_SQL_REVIEW);
  const on = useAppStore((state) => aiReviewEnabled(state.serverInfo));
  return licensed && on;
}
