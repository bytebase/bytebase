import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import {
  type ReviewRun,
  ReviewRun_Status,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { sqlReviewV2Enabled } from "@/utils/featureGates";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";

// The issue's AI review slot. The name addresses the slot, not one execution.
export const aiReviewRunName = (issueName: string): string =>
  `${issueName}/reviewRuns/ai`;

// Whether the plan page reads the issue's AI review run. The review is dark
// launched behind SQL Review V2, runs only while the workspace has AI and its
// review switch on, and reviews SQL that no rollout has frozen yet.
export const readsAIReviewRun = ({
  plan,
  project,
  serverInfo,
}: {
  plan: Plan;
  project: Project;
  serverInfo: ActuatorInfo | undefined;
}): boolean =>
  sqlReviewV2Enabled() &&
  Boolean(serverInfo?.aiEnabled && serverInfo.aiSqlReviewEnabled) &&
  Boolean(plan.issue) &&
  !plan.hasRollout &&
  hasProjectPermissionV2(project, "bb.reviewRuns.get");

// A queued or running execution; RunReview would replace it.
export const isAIReviewRunActive = (
  run: ReviewRun | null | undefined
): boolean =>
  run?.status === ReviewRun_Status.AVAILABLE ||
  run?.status === ReviewRun_Status.RUNNING;
