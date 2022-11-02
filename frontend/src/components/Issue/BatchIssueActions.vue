<template>
  <BBTooltipButton
    type="primary"
    :disabled="!isTransitionApplicableForAllIssues('RESOLVE')"
    tooltip-mode="DISABLED-ONLY"
    @click="doBatchIssueTransition('DONE')"
  >
    {{ $t("issue.batch-transition.resolve") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.resolved"),
          })
        }}
      </div>
    </template>
  </BBTooltipButton>

  <BBTooltipButton
    type="normal"
    :disabled="!isTransitionApplicableForAllIssues('CANCEL')"
    tooltip-mode="DISABLED-ONLY"
    @click="doBatchIssueTransition('CANCELED')"
  >
    {{ $t("issue.batch-transition.cancel") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.cancelled"),
          })
        }}
      </div>
    </template>
  </BBTooltipButton>

  <BBTooltipButton
    type="normal"
    :disabled="!isTransitionApplicableForAllIssues('REOPEN')"
    tooltip-mode="DISABLED-ONLY"
    @click="doBatchIssueTransition('OPEN')"
  >
    {{ $t("issue.batch-transition.reopen") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.reopened"),
          })
        }}
      </div>
    </template>
  </BBTooltipButton>

  <div
    v-if="state.isRequesting"
    class="fixed inset-0 bg-white/70 flex flex-col items-center justify-center gap-y-2"
  >
    <BBSpin />
    <div class="flex items-center textlabel">
      <span>{{ $t("common.updating") }}</span>
      <span v-if="state.stats"
        >({{ state.stats.success }} / {{ state.stats.total }})</span
      >
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";

import type {
  Issue,
  IssueStatus,
  IssueStatusPatch,
  IssueStatusTransition,
  IssueStatusTransitionType,
} from "@/types";
import {
  ASSIGNEE_APPLICABLE_ACTION_LIST,
  CREATOR_APPLICABLE_ACTION_LIST,
  ISSUE_STATUS_TRANSITION_LIST,
  SYSTEM_BOT_ID,
} from "@/types";
import { allTaskList, hasWorkspacePermission } from "@/utils";
import { refreshIssueList, useCurrentUser, useIssueStore } from "@/store";

type RequestStats = {
  total: number;
  success: number;
  failed: number;
};

type LocalState = {
  isRequesting: boolean;
  stats?: RequestStats;
};

const props = defineProps({
  issueList: {
    type: Array as PropType<Issue[]>,
    default: () => [],
  },
});

const state = reactive<LocalState>({
  isRequesting: false,
});

const currentUser = useCurrentUser();
const issueStore = useIssueStore();

const getApplicableIssueStatusTransitionList = (
  issue: Issue
): IssueStatusTransition[] => {
  const actionList: IssueStatusTransitionType[] = [];

  // The current user is the assignee of the issue
  // or the assignee is SYSTEM_BOT and the current user can manage issue
  const isAssignee =
    currentUser.value.id === issue.assignee?.id ||
    (issue.assignee?.id == SYSTEM_BOT_ID &&
      hasWorkspacePermission(
        "bb.permission.workspace.manage-issue",
        currentUser.value.role
      ));
  const isCreator = currentUser.value.id === issue.creator.id;
  if (isAssignee) {
    actionList.push(...ASSIGNEE_APPLICABLE_ACTION_LIST.get(issue.status)!);
  }
  if (isCreator) {
    CREATOR_APPLICABLE_ACTION_LIST.get(issue.status)!.forEach((item) => {
      if (actionList.indexOf(item) === -1) {
        actionList.push(item);
      }
    });
  }

  const applicableActionList: IssueStatusTransition[] = [];

  actionList.forEach((type) => {
    const transition = ISSUE_STATUS_TRANSITION_LIST.get(type)!;
    const taskList = allTaskList(issue.pipeline);
    if (taskList.some((task) => task.status === "RUNNING")) {
      // Disallow any issue status transition if some of the tasks are in RUNNING state.
      return;
    }
    if (type === "RESOLVE") {
      // Disallow to "resolve" issue if some of the tasks are NOT DONE.
      if (taskList.some((task) => task.status !== "DONE")) {
        return;
      }
    }
    applicableActionList.push(transition);
  });

  return applicableActionList;
};

const issueTransitionList = computed(() => {
  return props.issueList.map((issue) => {
    const transitions = getApplicableIssueStatusTransitionList(issue);
    return { issue, transitions };
  });
});

const isTransitionApplicableForAllIssues = (
  type: IssueStatusTransitionType
): boolean => {
  return issueTransitionList.value.every((item) => {
    return (
      item.transitions.findIndex((transition) => transition.type === type) >= 0
    );
  });
};

const doBatchIssueTransition = async (to: IssueStatus) => {
  const issueStatusPatch: IssueStatusPatch = {
    status: to,
    comment: "", // TODO: provide a dialog to input the comments
  };

  const stats = {
    total: props.issueList.length,
    success: 0,
    failed: 0,
  };

  const doSingleIssueTransition = (issue: Issue) => {
    const request = issueStore.updateIssueStatus({
      issueId: issue.id,
      issueStatusPatch,
    });
    request.then(
      () => stats.success++,
      () => stats.failed++
    );

    return request;
  };

  state.isRequesting = true;
  state.stats = stats;
  try {
    const requestList = props.issueList.map(doSingleIssueTransition);
    await Promise.allSettled(requestList);
  } finally {
    state.isRequesting = false;
    refreshIssueList();
  }
};
</script>
