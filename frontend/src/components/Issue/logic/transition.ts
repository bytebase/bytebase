import { computed, Ref } from "vue";
import { useCurrentUser } from "@/store";
import {
  Issue,
  SYSTEM_BOT_ID,
  ONBOARDING_ISSUE_ID,
  IssueStatusTransitionType,
  ASSIGNEE_APPLICABLE_ACTION_LIST,
  CREATOR_APPLICABLE_ACTION_LIST,
  ISSUE_STATUS_TRANSITION_LIST,
  IssueStatusTransition,
} from "@/types";
import {
  allTaskList,
  applicableTaskTransition,
  isDBAOrOwner,
  TaskStatusTransition,
} from "@/utils";
import { useIssueLogic } from ".";

export const useIssueTransitionLogic = (issue: Ref<Issue>) => {
  const { activeTaskOfPipeline } = useIssueLogic();
  const currentUser = useCurrentUser();

  const getApplicableIssueStatusTransitionList = (
    issue: Issue
  ): IssueStatusTransition[] => {
    const currentTask = activeTaskOfPipeline(issue.pipeline);

    const issueEntity = issue as Issue;
    if (issueEntity.id == ONBOARDING_ISSUE_ID) {
      return [];
    }
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
      .filter((item) => {
        const pipeline = issueEntity.pipeline;
        // Disallow any issue status transition if the active task is in RUNNING state.
        if (currentTask.status == "RUNNING") {
          return false;
        }

        const taskList = allTaskList(pipeline);
        // Don't display the Resolve action if the last task is NOT in DONE status.
        if (
          item == "RESOLVE" &&
          taskList.length > 0 &&
          (currentTask.id != taskList[taskList.length - 1].id ||
            currentTask.status != "DONE")
        ) {
          return false;
        }

        return true;
      })
      .map(
        (type: IssueStatusTransitionType) =>
          ISSUE_STATUS_TRANSITION_LIST.get(type)!
      )
      .reverse();
  };

  const getApplicableTaskStatusTransitionList = (
    issue: Issue
  ): TaskStatusTransition[] => {
    if (issue.id == ONBOARDING_ISSUE_ID) {
      return [];
    }
    switch (issue.status) {
      case "DONE":
      case "CANCELED":
        return [];
      case "OPEN": {
        let list: TaskStatusTransition[] = [];

        // Allow assignee, or assignee is the system bot and current user is DBA or owner
        if (
          currentUser.value.id === issue.assignee?.id ||
          (issue.assignee?.id == SYSTEM_BOT_ID &&
            isDBAOrOwner(currentUser.value.role))
        ) {
          list = applicableTaskTransition(issue.pipeline);
        }

        return list;
      }
    }
    console.assert(false, "Should never reach this line");
  };

  const applicableTaskStatusTransitionList = computed(() =>
    getApplicableTaskStatusTransitionList(issue.value)
  );

  const applicableIssueStatusTransitionList = computed(() =>
    getApplicableIssueStatusTransitionList(issue.value)
  );

  return {
    getApplicableIssueStatusTransitionList,
    getApplicableTaskStatusTransitionList,
    applicableIssueStatusTransitionList,
    applicableTaskStatusTransitionList,
  };
};

export function isApplicableTransition<
  T extends IssueStatusTransition | TaskStatusTransition
>(target: T, list: T[]): boolean {
  return (
    list.findIndex((applicable) => {
      return applicable.to === target.to && applicable.type === target.type;
    }) >= 0
  );
}
