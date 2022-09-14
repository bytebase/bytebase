import { computed, Ref } from "vue";
import { useCurrentUser } from "@/store";
import {
  Issue,
  SYSTEM_BOT_ID,
  IssueStatusTransitionType,
  ASSIGNEE_APPLICABLE_ACTION_LIST,
  CREATOR_APPLICABLE_ACTION_LIST,
  ISSUE_STATUS_TRANSITION_LIST,
  IssueStatusTransition,
} from "@/types";
import {
  allTaskList,
  applicableStageTransition,
  applicableTaskTransition,
  isDBAOrOwner,
  isOwnerOfProject,
  StageStatusTransition,
  TaskStatusTransition,
} from "@/utils";
import { useAllowProjectOwnerToApprove, useIssueLogic } from ".";

export const useIssueTransitionLogic = (issue: Ref<Issue>) => {
  const {
    create,
    activeTaskOfPipeline,
    allowApplyIssueStatusTransition,
    allowApplyTaskStatusTransition,
  } = useIssueLogic();

  const currentUser = useCurrentUser();

  const allowProjectOwnerAsAssignee = useAllowProjectOwnerToApprove();

  const isAllowedToApplyTaskTransition = computed(() => {
    if (create.value) {
      return false;
    }

    // Applying task transition is decoupled with the issue's Assignee.
    // But relative to the task's environment's approval policy.
    if (isDBAOrOwner(currentUser.value.role)) {
      return true;
    }
    if (
      allowProjectOwnerAsAssignee.value &&
      isOwnerOfProject(currentUser.value, (issue.value as Issue).project)
    ) {
      return true;
    }
    return false;
  });

  const getApplicableIssueStatusTransitionList = (
    issue: Issue
  ): IssueStatusTransition[] => {
    if (create.value) {
      return [];
    }

    const currentTask = activeTaskOfPipeline(issue.pipeline);

    const issueEntity = issue as Issue;
    const list: IssueStatusTransitionType[] = [];
    // Allow assignee, or assignee is the system bot and current user is DBA or owner
    if (
      currentUser.value.id === issueEntity.assignee?.id ||
      (issueEntity.assignee?.id == SYSTEM_BOT_ID &&
        isDBAOrOwner(currentUser.value.role))
    ) {
      list.push(...ASSIGNEE_APPLICABLE_ACTION_LIST.get(issueEntity.status)!);
    }
    if (currentUser.value.id === issueEntity.creator.id) {
      CREATOR_APPLICABLE_ACTION_LIST.get(issueEntity.status)!.forEach(
        (item) => {
          if (list.indexOf(item) == -1) {
            list.push(item);
          }
        }
      );
    }

    return list
      .map(
        (type: IssueStatusTransitionType) =>
          ISSUE_STATUS_TRANSITION_LIST.get(type)!
      )
      .filter((transition) => {
        const pipeline = issueEntity.pipeline;
        // Disallow any issue status transition if the active task is in RUNNING state.
        if (currentTask.status == "RUNNING") {
          return false;
        }

        const taskList = allTaskList(pipeline);
        // Don't display the Resolve action if the last task is NOT in DONE status.
        if (
          transition.type == "RESOLVE" &&
          taskList.length > 0 &&
          (currentTask.id != taskList[taskList.length - 1].id ||
            currentTask.status != "DONE")
        ) {
          return false;
        }

        return allowApplyIssueStatusTransition(issue, transition.to);
      })
      .reverse();
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
          const currentTask = activeTaskOfPipeline(issue.pipeline);
          return applicableStageTransition(issue.pipeline).filter(
            (transition) =>
              allowApplyTaskStatusTransition(currentTask, transition.to)
          );
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

export function isApplicableTransition<
  T extends IssueStatusTransition | TaskStatusTransition | StageStatusTransition
>(target: T, list: T[]): boolean {
  return (
    list.findIndex((applicable) => {
      return applicable.to === target.to && applicable.type === target.type;
    }) >= 0
  );
}
