import { isValidDatabaseGroupName } from "@/types/dbGroup";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import {
  type DatabaseGroup,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  type ReviewRun,
  ReviewRun_State,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

// The issue's review run. The name addresses the current run, which a re-run
// replaces.
export const reviewRunName = (issueName: string): string =>
  `${issueName}/reviewRun`;

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

// Whether a review run includes the AI review: the workspace has AI and its
// review switch on. Such a run makes model calls.
export const aiReviewEnabled = (serverInfo: ActuatorInfo | undefined) =>
  Boolean(serverInfo?.aiEnabled && serverInfo.aiSqlReviewEnabled);

// A queued or running execution; RunReview would replace it.
export const isReviewRunActive = (run: ReviewRun | null | undefined) =>
  run?.state === ReviewRun_State.RUNNING;
