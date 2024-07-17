import { useCurrentUserV1 } from "@/store";
import type {
  ComposedIssue,
  IssueStatusTransitionType,
  IssueStatusTransition,
  ComposedUser,
} from "@/types";
import { ISSUE_STATUS_TRANSITION_LIST } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import type { StageStatusTransition, TaskStatusTransition } from "@/utils";
import {
  isDatabaseChangeRelatedIssue,
  flattenTaskV1List,
  isGrantRequestIssue,
  activeTaskInRollout,
  hasProjectPermissionV2,
} from "@/utils";
import { extractReviewContext } from "./review";

// The first transition in the list is the primary action and the rests are
// the normal action. For now there are at most 1 primary 1 normal action.
const APPLICABLE_ISSUE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  [IssueStatus.OPEN, ["RESOLVE", "CANCEL"]],
  [IssueStatus.DONE, ["REOPEN"]],
  [IssueStatus.CANCELED, ["REOPEN"]],
]);

export const calcApplicableIssueStatusTransitionList = (
  issue: ComposedIssue
): IssueStatusTransition[] => {
  const transitionTypeList: IssueStatusTransitionType[] = [];
  const currentUserV1 = useCurrentUserV1();

  if (allowUserToApplyIssueStatusTransition(issue, currentUserV1.value)) {
    const actions = APPLICABLE_ISSUE_ACTION_LIST.get(issue.status);
    if (actions) {
      transitionTypeList.push(...actions);
    }
  }

  const applicableTransitionList: IssueStatusTransition[] = [];
  transitionTypeList.forEach((type) => {
    const transition = ISSUE_STATUS_TRANSITION_LIST.get(type);
    if (!transition) return;

    if (type === "RESOLVE") {
      // If an issue is not "Approved" in review stage
      // it cannot be Resolved.
      if (!isIssueReviewDone(issue)) {
        return;
      }
    } else if (type === "REOPEN") {
      // Don't show the REOPEN button for request granting issues.
      if (isGrantRequestIssue(issue)) {
        return;
      }
    }

    if (isDatabaseChangeRelatedIssue(issue)) {
      const currentTask = activeTaskInRollout(issue.rolloutEntity);
      const flattenTaskList = flattenTaskV1List(issue.rolloutEntity);
      if (flattenTaskList.some((task) => task.status === Task_Status.RUNNING)) {
        // Disallow any issue status transition if some of the tasks are in RUNNING state.
        return;
      }
      if (type === "RESOLVE") {
        if (transition.type === "RESOLVE" && flattenTaskList.length > 0) {
          const lastTask = flattenTaskList[flattenTaskList.length - 1];
          // Don't display the RESOLVE action if the pipeline doesn't reach the
          // last task
          if (lastTask.uid !== currentTask.uid) {
            return;
          }
          // Don't display the RESOLVE action if the last task is not DONE.
          if (currentTask.status !== Task_Status.DONE) {
            return;
          }
        }
      }
    }
    applicableTransitionList.push(transition);
  });
  return applicableTransitionList;
};

export function isApplicableTransition<
  T extends
    | IssueStatusTransition
    | TaskStatusTransition
    | StageStatusTransition,
>(target: T, list: T[]): boolean {
  return (
    list.findIndex((applicable) => {
      return applicable.to === target.to && applicable.type === target.type;
    }) >= 0
  );
}

const allowUserToApplyIssueStatusTransition = (
  issue: ComposedIssue,
  user: ComposedUser
) => {
  // Allowed if the user has issues.update permission in the project
  if (hasProjectPermissionV2(issue.projectEntity, user, "bb.issues.update")) {
    return true;
  }

  // The creator can apply issue status transition
  if (user.name === issue.creatorEntity.name) {
    return true;
  }

  return false;
};

function isIssueReviewDone(issue: ComposedIssue) {
  const context = extractReviewContext(issue);
  return context.done.value;
}
