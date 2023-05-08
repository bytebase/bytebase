import { computed, Ref } from "vue";
import { useCurrentUser } from "@/store";
import {
  Issue,
  IssueStatusTransitionType,
  ISSUE_STATUS_TRANSITION_LIST,
  IssueStatusTransition,
  APPLICABLE_ISSUE_ACTION_LIST,
} from "@/types";
import {
  activeStage,
  activeTask,
  allTaskList,
  applicableTaskTransition,
  StageStatusTransition,
  TaskStatusTransition,
  TASK_STATUS_TRANSITION_LIST,
} from "@/utils";
import {
  allowUserToBeAssignee,
  useCurrentRollOutPolicyForActiveEnvironment,
} from "./";
import { useIssueLogic } from ".";

export const useIssueTransitionLogic = (issue: Ref<Issue>) => {
  const { create, activeTaskOfPipeline, allowApplyTaskStatusTransition } =
    useIssueLogic();

  const currentUser = useCurrentUser();
  const rollOutPolicy = useCurrentRollOutPolicyForActiveEnvironment();

  const isAllowedToApplyTaskTransition = computed(() => {
    if (create.value) {
      return false;
    }

    if (
      allowUserToBeAssignee(
        currentUser.value,
        issue.value.project,
        rollOutPolicy.value.policy,
        rollOutPolicy.value.assigneeGroup
      )
    ) {
      return true;
    }

    // Otherwise, only the assignee can apply task status transitions
    // including roll out, cancel, retry, etc.
    return issue.value.assignee.id === currentUser.value.id;
  });

  const getApplicableIssueStatusTransitionList = (
    issue: Issue
  ): IssueStatusTransition[] => {
    if (create.value) {
      return [];
    }
    return calcApplicableIssueStatusTransitionList(issue);
  };

  const getApplicableStageStatusTransitionList = (issue: Issue) => {
    if (create.value) {
      return [];
    }
    switch (issue.status) {
      case "DONE":
      case "CANCELED":
        return [];
      case "OPEN": {
        if (isAllowedToApplyTaskTransition.value) {
          // Only "Approve" can be applied to current stage by now.
          const ROLLOUT = TASK_STATUS_TRANSITION_LIST.get("ROLLOUT")!;
          const currentStage = activeStage(issue.pipeline);

          const pendingApprovalTaskList = currentStage.taskList.filter(
            (task) => {
              return (
                task.status === "PENDING_APPROVAL" &&
                allowApplyTaskStatusTransition(task, ROLLOUT.to)
              );
            }
          );

          // Allowing "Approve" a stage when it has TWO OR MORE tasks
          // are "PENDING_APPROVAL" (including the "activeTask" itself)
          if (pendingApprovalTaskList.length >= 2) {
            return [ROLLOUT];
          }
        }

        return [];
      }
    }
    console.assert(false, "Should never reach this line");
  };

  const getApplicableTaskStatusTransitionList = (
    issue: Issue
  ): TaskStatusTransition[] => {
    if (create.value) {
      return [];
    }
    switch (issue.status) {
      case "DONE":
      case "CANCELED":
        return [];
      case "OPEN": {
        if (isAllowedToApplyTaskTransition.value) {
          const currentTask = activeTaskOfPipeline(issue.pipeline);
          return applicableTaskTransition(issue.pipeline).filter((transition) =>
            allowApplyTaskStatusTransition(currentTask, transition.to)
          );
        }

        return [];
      }
    }
    console.assert(false, "Should never reach this line");
  };

  const applicableTaskStatusTransitionList = computed(() =>
    getApplicableTaskStatusTransitionList(issue.value)
  );

  const applicableStageStatusTransitionList = computed(() =>
    getApplicableStageStatusTransitionList(issue.value)
  );

  const applicableIssueStatusTransitionList = computed(() =>
    getApplicableIssueStatusTransitionList(issue.value)
  );

  return {
    getApplicableIssueStatusTransitionList,
    getApplicableStageStatusTransitionList,
    getApplicableTaskStatusTransitionList,
    applicableIssueStatusTransitionList,
    applicableStageStatusTransitionList,
    applicableTaskStatusTransitionList,
  };
};

export const calcApplicableIssueStatusTransitionList = (
  issue: Issue
): IssueStatusTransition[] => {
  const currentUser = useCurrentUser();
  const currentTask = activeTask(issue.pipeline);
  const flattenTaskList = allTaskList(issue.pipeline);

  const issueEntity = issue as Issue;
  const transitionTypeList: IssueStatusTransitionType[] = [];

  // The creator and the assignee can apply issue status transition
  // including resolve, cancel, reopen
  if (
    currentUser.value.id === issueEntity.creator?.id ||
    currentUser.value.id === issueEntity.assignee?.id
  ) {
    const actions = APPLICABLE_ISSUE_ACTION_LIST.get(issueEntity.status);
    if (actions) {
      transitionTypeList.push(...actions);
    }
  }

  const applicableTransitionList: IssueStatusTransition[] = [];
  transitionTypeList.forEach((type) => {
    const transition = ISSUE_STATUS_TRANSITION_LIST.get(type);
    if (!transition) return;

    if (flattenTaskList.some((task) => task.status === "RUNNING")) {
      // Disallow any issue status transition if some of the tasks are in RUNNING state.
      return;
    }
    if (type === "RESOLVE") {
      if (transition.type === "RESOLVE" && flattenTaskList.length > 0) {
        const lastTask = flattenTaskList[flattenTaskList.length - 1];
        // Don't display the RESOLVE action if the pipeline doesn't reach the
        // last task
        if (lastTask.id !== currentTask.id) {
          return;
        }
        // Don't display the RESOLVE action if the last task is not DONE.
        if (currentTask.status !== "DONE") {
          return;
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
