import { isValidDatabaseGroupName } from "@/types/dbGroup";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import {
  type DatabaseGroup,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
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

// How many databases a database group matches, or undefined until the
// page has loaded the group's full view.
export type MatchedDatabaseCount = (group: string) => number | undefined;

export const loadedMatchedDatabaseCount =
  (state: {
    dbGroupsByName: Record<string, DatabaseGroup>;
    dbGroupViewByName: Record<string, DatabaseGroupView>;
  }): MatchedDatabaseCount =>
  (group) =>
    state.dbGroupViewByName[group] === DatabaseGroupView.FULL
      ? (state.dbGroupsByName[group]?.matchedDatabases.length ?? 0)
      : undefined;

// Mirrors the backend's DeriveReviewTargets, and RunReview refuses a plan it
// finds nothing in: create-database and release-backed specs carry no SQL,
// and a spec whose one target is a database group reviews the group's
// matched databases. A group not loaded yet counts as reviewable.
export const hasReviewableSQL = (
  plan: Plan,
  matchedDatabaseCount: MatchedDatabaseCount
): boolean =>
  plan.specs.some((spec) => {
    if (
      spec.config.case !== "changeDatabaseConfig" ||
      spec.config.value.release
    ) {
      return false;
    }
    const { targets } = spec.config.value;
    if (targets.length === 1 && isValidDatabaseGroupName(targets[0])) {
      return (matchedDatabaseCount(targets[0]) ?? 1) > 0;
    }
    return targets.length > 0;
  });

// Whether the plan page reads the issue's AI review run. The review is dark
// launched behind SQL Review V2, runs only while the workspace has AI and its
// review switch on, and reviews SQL that no rollout has frozen yet.
export const readsAIReviewRun = ({
  plan,
  project,
  serverInfo,
  matchedDatabaseCount,
}: {
  plan: Plan;
  project: Project;
  serverInfo: ActuatorInfo | undefined;
  matchedDatabaseCount: MatchedDatabaseCount;
}): boolean =>
  sqlReviewV2Enabled() &&
  Boolean(serverInfo?.aiEnabled && serverInfo.aiSqlReviewEnabled) &&
  Boolean(plan.issue) &&
  !plan.hasRollout &&
  hasReviewableSQL(plan, matchedDatabaseCount) &&
  hasProjectPermissionV2(project, "bb.reviewRuns.get");

// A queued or running execution; RunReview would replace it.
export const isAIReviewRunActive = (
  run: ReviewRun | null | undefined
): boolean =>
  run?.status === ReviewRun_Status.AVAILABLE ||
  run?.status === ReviewRun_Status.RUNNING;
