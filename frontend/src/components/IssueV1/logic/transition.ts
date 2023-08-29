import { computed } from "vue";
import { extractIssueReviewContext } from "@/plugins/issue/logic";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import {
  ComposedIssue,
  IssueStatusTransitionType,
  ISSUE_STATUS_TRANSITION_LIST,
  IssueStatusTransition,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import {
  StageStatusTransition,
  TaskStatusTransition,
  hasWorkspacePermissionV1,
  isOwnerOfProjectV1,
  isDatabaseRelatedIssue,
  flattenTaskV1List,
  isGrantRequestIssue,
  activeTaskInRollout,
} from "@/utils";

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

    if (isDatabaseRelatedIssue(issue)) {
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
  T extends IssueStatusTransition | TaskStatusTransition | StageStatusTransition
>(target: T, list: T[]): boolean {
  return (
    list.findIndex((applicable) => {
      return applicable.to === target.to && applicable.type === target.type;
    }) >= 0
  );
}

const allowUserToApplyIssueStatusTransition = (
  issue: ComposedIssue,
  user: User
) => {
  // Workspace level high-privileged user (DBA/OWNER) are always allowed.
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      user.userRole
    )
  ) {
    return true;
  }

  // Project owners are also allowed
  const projectV1 = useProjectV1Store().getProjectByName(issue.project);
  if (isOwnerOfProjectV1(projectV1.iamPolicy, user)) {
    return true;
  }

  // The creator and the assignee can apply issue status transition

  if (user.name === issue.creatorEntity.name) {
    return true;
  }
  if (user.name === issue.assigneeEntity?.name) {
    return true;
  }

  return false;
};

function isIssueReviewDone(issue: ComposedIssue) {
  const context = extractIssueReviewContext(
    computed(() => undefined),
    computed(() => issue)
  );
  return context.done.value;
}
