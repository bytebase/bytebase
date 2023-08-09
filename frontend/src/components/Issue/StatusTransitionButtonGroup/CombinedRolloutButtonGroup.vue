<template>
  <template v-if="create">
    <button
      type="button"
      class="btn-primary px-4 py-2"
      :disabled="!allowCreate"
      data-label="bb-issue-create-button"
      @click.prevent="doCreate"
    >
      {{ $t("common.create") }}
    </button>
  </template>
  <template v-else>
    <div class="flex space-x-2">
      <TaskStatusTransitionButtonGroup
        v-if="applicableTaskStatusTransitionList.length > 0"
        :issue-context="issueContext"
        :task-status-transition-list="applicableTaskStatusTransitionList"
        :stage-status-transition-list="applicableStageStatusTransitionList"
        @apply-task-transition="tryStartStageOrTaskStatusTransition"
      />
      <IssueStatusTransitionButtonGroup
        :display-mode="
          applicableTaskStatusTransitionList.length === 0
            ? 'BUTTON'
            : 'DROPDOWN'
        "
        :issue-context="issueContext"
        :extra-action-list="extraActionList"
        :issue-status-transition-list="issueStatusTransitionActionList"
        @apply-issue-transition="tryStartIssueStatusTransition"
        @apply-batch-task-transition="tryStartBatchTaskTransition"
      />
    </div>

    <TaskStatusTransitionDialog
      v-if="onGoingTaskOrStageStatusTransition && currentTask"
      :task="currentTask"
      :mode="onGoingTaskOrStageStatusTransition.mode"
      :transition="onGoingTaskOrStageStatusTransition.transition"
      @updated="onGoingTaskOrStageStatusTransition = undefined"
      @cancel="onGoingTaskOrStageStatusTransition = undefined"
    />

    <IssueStatusTransitionDialog
      v-if="onGoingIssueStatusTransition"
      :transition="onGoingIssueStatusTransition.transition"
      @updated="onGoingIssueStatusTransition = undefined"
      @cancel="onGoingIssueStatusTransition = undefined"
    />

    <BatchTaskActionDialog
      v-if="onGoingBatchTaskStatusTransition"
      :transition="onGoingBatchTaskStatusTransition.transition"
      :task-list="onGoingBatchTaskStatusTransition.taskList"
      @updated="onGoingBatchTaskStatusTransition = undefined"
      @cancel="onGoingBatchTaskStatusTransition = undefined"
    />
  </template>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty } from "lodash-es";
import { computed, ref, Ref } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { convertUserToPrincipal, useCurrentUserV1 } from "@/store";
import type {
  GrantRequestContext,
  Issue,
  IssueCreate,
  IssueStatusTransition,
  Task,
  TaskCreate,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  activeStage,
  canSkipTask,
  isDatabaseRelatedIssueType,
  isGrantRequestIssueType,
  StageStatusTransition,
  TASK_STATUS_TRANSITION_LIST,
  TaskStatusTransition,
} from "@/utils";
import {
  flattenTaskList,
  useIssueTransitionLogic,
  TaskTypeWithStatement,
  useIssueLogic,
} from "../logic";
import BatchTaskActionDialog from "./BatchTaskActionDialog.vue";
import IssueStatusTransitionButtonGroup from "./IssueStatusTransitionButtonGroup.vue";
import IssueStatusTransitionDialog from "./IssueStatusTransitionDialog.vue";
import TaskStatusTransitionButtonGroup from "./TaskStatusTransitionButtonGroup.vue";
import TaskStatusTransitionDialog from "./TaskStatusTransitionDialog.vue";
import { ExtraActionOption, IssueContext } from "./common";

const { t } = useI18n();

const {
  create,
  issue,
  template: issueTemplate,
  activeTaskOfPipeline,
  doCreate,
  allowApplyTaskStatusTransition,
} = useIssueLogic();

const onGoingIssueStatusTransition = ref<{
  transition: IssueStatusTransition;
}>();
const onGoingTaskOrStageStatusTransition = ref<{
  mode: "STAGE" | "TASK";
  transition: TaskStatusTransition | StageStatusTransition;
}>();
const onGoingBatchTaskStatusTransition = ref<{
  transition: TaskStatusTransition;
  taskList: Task[];
}>();

const currentUser = useCurrentUserV1();

const issueReview = useIssueReviewContext();
const { done: reviewDone } = issueReview;

const issueContext = computed((): IssueContext => {
  return {
    currentUser: convertUserToPrincipal(currentUser.value),
    create: create.value,
    issue: issue.value,
  };
});

const {
  applicableTaskStatusTransitionList,
  applicableStageStatusTransitionList,
  applicableIssueStatusTransitionList,
} = useIssueTransitionLogic(issue as Ref<Issue>);

const issueStatusTransitionActionList = computed(() => {
  const actionList = cloneDeep(applicableIssueStatusTransitionList.value);
  const resolveActionIndex = actionList.findIndex(
    (item) => item.type === "RESOLVE"
  );
  // Hide resolve button when grant request issue isn't review done.
  if (isGrantRequestIssueType(issue.value.type) && resolveActionIndex > -1) {
    if (!reviewDone.value) {
      actionList.splice(resolveActionIndex, 1);
    }
  }
  return actionList;
});

const retryableTaskList = computed(() => {
  if (create.value) return [];

  const issueEntity = issue.value as Issue;
  if (issueEntity.status !== "OPEN") {
    return [];
  }
  if (!isDatabaseRelatedIssueType(issueEntity.type)) {
    return [];
  }

  const currentStage = activeStage(issueEntity.pipeline!);
  const RETRY = TASK_STATUS_TRANSITION_LIST.get("RETRY")!;
  return currentStage.taskList.filter((task) => {
    return (
      task.status === "FAILED" && allowApplyTaskStatusTransition(task, RETRY.to)
    );
  });
});

const skippableTaskList = computed(() => {
  if (create.value) return [];

  const issueEntity = issue.value as Issue;
  if (issueEntity.status !== "OPEN") {
    return [];
  }
  if (!isDatabaseRelatedIssueType(issueEntity.type)) {
    return [];
  }

  const currentStage = activeStage(issueEntity.pipeline!);
  return currentStage.taskList.filter((task) =>
    canSkipTask(
      task,
      issueEntity,
      false /* !activeOnly */,
      true /* failedOnly */
    )
  );
});

const extraActionList = computed(() => {
  const list: ExtraActionOption[] = [];

  if (skippableTaskList.value.length > 0) {
    list.push({
      label: t("task.skip-failed-in-current-stage"),
      key: "skip-failed-tasks-in-current-stage",
      type: "TASK-BATCH",
      transition: {
        buttonType: "PRIMARY",
        buttonName: t("task.skip"),
        type: "SKIP",
        to: "DONE",
      },
      target: skippableTaskList.value,
    });
  }

  return list;
});

const tryStartBatchTaskTransition = (
  transition: TaskStatusTransition,
  taskList: Task[]
) => {
  if (transition.type === "SKIP") {
    onGoingBatchTaskStatusTransition.value = {
      transition,
      taskList,
    };
  }
};

const tryStartStageOrTaskStatusTransition = (
  transition: TaskStatusTransition | StageStatusTransition,
  mode: "STAGE" | "TASK"
) => {
  if (transition.type === "RETRY" && mode === "STAGE") {
    // RETRYing current stage won't use stage status transition API endpoint.
    // Use batch task status transition instead.
    const taskList = retryableTaskList.value;
    onGoingBatchTaskStatusTransition.value = {
      transition,
      taskList,
    };
    return;
  }

  onGoingTaskOrStageStatusTransition.value = {
    mode,
    transition,
  };
};

const currentTask = computed(() => {
  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return undefined;
  }
  return activeTaskOfPipeline((issue.value as Issue).pipeline!);
});

const tryStartIssueStatusTransition = (transition: IssueStatusTransition) => {
  onGoingIssueStatusTransition.value = {
    transition,
  };
};

const allowCreate = computed(() => {
  const newIssue = issue.value as IssueCreate;

  if (isEmpty(newIssue.name)) {
    return false;
  }

  if (newIssue.assigneeId == UNKNOWN_ID) {
    return false;
  }

  if (isDatabaseRelatedIssueType(newIssue.type)) {
    const allTaskList = flattenTaskList<TaskCreate>(newIssue);
    for (const task of allTaskList) {
      if (TaskTypeWithStatement.includes(task.type)) {
        if (
          isEmpty(task.statement) &&
          ((task as TaskCreate).sheetId === undefined ||
            (task as TaskCreate).sheetId === UNKNOWN_ID)
        ) {
          return false;
        }
      }
    }
  } else if (isGrantRequestIssueType(newIssue.type)) {
    const createContext = newIssue.createContext as GrantRequestContext;
    if (createContext.role === "EXPORTER") {
      return (
        createContext.databaseResources.length > 0 &&
        createContext.statement !== ""
      );
    }
  }

  const template = issueTemplate.value;
  for (const field of template.inputFieldList) {
    if (
      field.type !== "Boolean" && // Switch is boolean value which always is present
      !field.resolved(issueContext.value)
    ) {
      return false;
    }
  }
  return true;
});
</script>
