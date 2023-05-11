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
    <div
      v-if="applicableTaskStatusTransitionList.length > 0"
      class="flex space-x-2"
    >
      <template
        v-for="(transition, index) in applicableTaskStatusTransitionList"
        :key="index"
      >
        <BBContextMenuButton
          data-label="bb-issue-status-transition-button"
          default-action-key="ROLLOUT-STAGE"
          :disabled="!allowApplyTaskTransition(transition)"
          :action-list="getButtonActionList(transition)"
          @click="(action) => onClickTaskStatusTransitionActionButton(action as TaskStatusTransitionButtonAction)"
        />
      </template>
      <template v-if="mergedDropdownActionList.length > 0">
        <NDropdown
          trigger="click"
          placement="bottom-end"
          :options="mergedDropdownActionList"
          @select="handleDropdownSelect"
        >
          <button
            id="user-menu"
            type="button"
            class="text-control-light p-0.5 rounded hover:bg-control-bg-hover"
            aria-label="User menu"
            aria-haspopup="true"
          >
            <heroicons-solid:dots-vertical class="w-6 h-6" />
          </button>
        </NDropdown>
      </template>
    </div>
    <template v-else>
      <div
        v-if="applicableIssueStatusTransitionList.length > 0"
        class="flex space-x-2"
      >
        <template
          v-for="(transition, index) in applicableIssueStatusTransitionList"
          :key="index"
        >
          <button
            type="button"
            :class="transition.buttonClass"
            :disabled="!allowIssueStatusTransition(transition)"
            @click.prevent="tryStartIssueStatusTransition(transition)"
          >
            {{ $t(transition.buttonName) }}
          </button>
        </template>

        <template v-if="extraActionList.length > 0">
          <NDropdown
            trigger="click"
            placement="bottom-end"
            :options="extraActionList"
            @select="handleDropdownSelect"
          >
            <button
              id="user-menu"
              type="button"
              class="text-control-light p-0.5 rounded hover:bg-control-bg-hover"
              aria-label="User menu"
              aria-haspopup="true"
            >
              <heroicons-solid:dots-vertical class="w-6 h-6" />
            </button>
          </NDropdown>
        </template>
      </div>
    </template>
  </template>
  <BBModal
    v-if="updateStatusModalState.show"
    :title="updateStatusModalState.title"
    class="relative overflow-hidden"
    @close="updateStatusModalState.show = false"
  >
    <div
      v-if="updateStatusModalState.isTransiting"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
    <StatusTransitionForm
      :mode="updateStatusModalState.mode"
      :ok-text="updateStatusModalState.okText"
      :issue="(issue as Issue)"
      :task="currentTask"
      :transition="updateStatusModalState.transition!"
      :output-field-list="issueTemplate.outputFieldList"
      @submit="onSubmit"
      @cancel="
        () => {
          updateStatusModalState.show = false;
        }
      "
    />
  </BBModal>

  <BBModal
    v-if="batchTaskActionState"
    :title="batchTaskActionState.title"
    class="relative overflow-hidden"
    @close="batchTaskActionState = undefined"
  >
    <BatchTaskActionForm
      :issue="(issue as Issue)"
      :transition="batchTaskActionState.transition"
      :task-list="batchTaskActionState.taskList"
      :ok-text="batchTaskActionState.transition.buttonName"
      :title="batchTaskActionState.title"
      @cancel="batchTaskActionState = undefined"
      @finish="batchTaskActionState = undefined"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, Ref } from "vue";
import { isEmpty } from "lodash-es";
import { useI18n } from "vue-i18n";
import { DropdownOption, NDropdown } from "naive-ui";

import {
  activeStage,
  canSkipTask,
  isDatabaseRelatedIssueType,
  StageStatusTransition,
  taskCheckRunSummary,
  TaskStatusTransition,
} from "@/utils";
import type {
  Issue,
  IssueCreate,
  IssueStatusTransition,
  Principal,
  Stage,
  Task,
  TaskCreate,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useCurrentUser, useIssueStore } from "@/store";
import StatusTransitionForm from "./StatusTransitionForm.vue";
import {
  flattenTaskList,
  useIssueTransitionLogic,
  isApplicableTransition,
  IssueTypeWithStatement,
  TaskTypeWithStatement,
  useExtraIssueLogic,
  useIssueLogic,
} from "./logic";
import BBContextMenuButton, {
  type ButtonAction,
} from "@/bbkit/BBContextMenuButton.vue";
import BatchTaskActionForm from "./BatchTaskActionForm.vue";

export type IssueContext = {
  currentUser: Principal;
  create: boolean;
  issue: Issue | IssueCreate;
};

interface UpdateStatusModalState {
  mode: "ISSUE" | "STAGE" | "TASK";
  show: boolean;
  style: string;
  okText: string;
  title: string;
  transition?:
    | IssueStatusTransition
    | StageStatusTransition
    | TaskStatusTransition;
  payload?: Task | Stage;
  isTransiting: boolean;
}

type BatchTaskActionState = {
  taskList: Task[];
  transition: TaskStatusTransition;
  title: string;
  okText: string;
};

type TaskStatusTransitionButtonAction = ButtonAction<{
  transition: TaskStatusTransition;
  target: "TASK" | "STAGE";
}>;

type ExtraAction<T extends "ISSUE" | "TASK" | "TASK-BATCH"> = {
  type: T;
  transition: T extends "ISSUE" ? IssueStatusTransition : TaskStatusTransition;
  target: T extends "ISSUE"
    ? Issue
    : T extends "TASK"
    ? Task
    : T extends "TASK-BATCH"
    ? Task[]
    : unknown;
};

type ExtraActionOption = DropdownOption &
  ExtraAction<"ISSUE" | "TASK" | "TASK-BATCH">;

const { t } = useI18n();

const {
  create,
  issue,
  template: issueTemplate,
  activeTaskOfPipeline,
  allowApplyTaskStatusTransition,
  doCreate,
} = useIssueLogic();
const { changeIssueStatus, changeStageAllTaskStatus, changeTaskStatus } =
  useExtraIssueLogic();

const updateStatusModalState = reactive<UpdateStatusModalState>({
  mode: "ISSUE",
  show: false,
  style: "INFO",
  okText: "OK",
  title: "",
  isTransiting: false,
});

const batchTaskActionState = ref<BatchTaskActionState>();

const currentUser = useCurrentUser();

const issueContext = computed((): IssueContext => {
  return {
    currentUser: currentUser.value,
    create: create.value,
    issue: issue.value,
  };
});

const {
  applicableTaskStatusTransitionList,
  applicableStageStatusTransitionList,
  applicableIssueStatusTransitionList,
  getApplicableIssueStatusTransitionList,
  getApplicableStageStatusTransitionList,
  getApplicableTaskStatusTransitionList,
} = useIssueTransitionLogic(issue as Ref<Issue>);

// TODO(steven): figure out how to show the dropdown for grant request issue.
const issueStatusTransitionDropdownOptions = computed(() => {
  return applicableIssueStatusTransitionList.value.map<ExtraActionOption>(
    (transition) => {
      return {
        key: transition.type,
        label: t(transition.buttonName),
        type: "ISSUE",
        transition,
        target: issue.value as Issue,
      };
    }
  );
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

  const currentStage = activeStage(issueEntity.pipeline);
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

const mergedDropdownActionList = computed(() => {
  if (applicableTaskStatusTransitionList.value.length > 0) {
    // When there are something to do with tasks, they will be shown as big
    // buttons.
    // Now we display issue-level actions as a dropdown together with "extra"
    // actions.
    return [
      ...issueStatusTransitionDropdownOptions.value,
      ...extraActionList.value,
    ];
  } else {
    // When we have nothing to do with tasks, show issue-level actions as big
    // buttons. And show only "extra" actions as a dropdown.
    return [...extraActionList.value];
  }
});

const handleDropdownSelect = (key: string, option: ExtraActionOption) => {
  if (option.type === "ISSUE") {
    tryStartIssueStatusTransition(option.transition as IssueStatusTransition);
  }
  if (option.type === "TASK-BATCH") {
    tryStartBatchTaskTransition(
      option.transition as TaskStatusTransition,
      option.target as Task[]
    );
  }
};

const tryStartBatchTaskTransition = (
  transition: TaskStatusTransition,
  taskList: Task[]
) => {
  if (transition.type === "SKIP") {
    batchTaskActionState.value = {
      okText: transition.buttonName,
      title: t("task.skip-failed-in-current-stage"),
      transition,
      taskList,
    };
  }
};

const tryStartStageOrTaskStatusTransition = (
  transition: TaskStatusTransition | StageStatusTransition,
  mode: "STAGE" | "TASK"
) => {
  updateStatusModalState.mode = mode;
  const task = currentTask.value;
  const payload = mode === "TASK" ? task : task.stage;
  const type = mode === "TASK" ? t("common.task") : t("common.stage");
  const name = payload.name;
  let actionText = "";
  switch (transition.type) {
    case "RUN":
      actionText = t("common.run");
      break;
    case "ROLLOUT":
      actionText = t("common.rollout");
      break;
    case "RETRY":
      actionText = t("common.retry");
      break;
    case "CANCEL":
      actionText = t("common.cancel");
      break;
    case "SKIP":
      actionText = t("common.skip");
      break;
    case "RESTART":
      actionText = t("common.restart");
      break;
    default:
      console.assert(false, "should never reach this line");
  }
  updateStatusModalState.title = t("issue.status-transition.modal.title", {
    action: actionText,
    type: type.toLowerCase(),
    name,
  });
  const button = t(transition.buttonName);
  if (mode === "TASK") {
    updateStatusModalState.okText = button;
  } else {
    const pendingApprovalTaskList = task.stage.taskList.filter((task) => {
      return (
        task.status === "PENDING_APPROVAL" &&
        allowApplyTaskStatusTransition(task, "PENDING")
      );
    });
    updateStatusModalState.okText = t(
      "issue.status-transition.modal.action-to-stage",
      {
        action: button,
        n: pendingApprovalTaskList.length,
      }
    );
  }
  updateStatusModalState.style = "INFO";
  updateStatusModalState.transition = transition;
  updateStatusModalState.payload = payload;
  updateStatusModalState.show = true;
};

const doTaskStatusTransition = (
  transition: TaskStatusTransition,
  task: Task,
  comment: string
) => {
  changeTaskStatus(task, transition.to, comment);
};

const doStageStatusTransition = (
  transition: StageStatusTransition,
  stage: Stage,
  comment: string
) => {
  changeStageAllTaskStatus(stage, transition.to, comment);
};

const currentTask = computed(() => {
  return activeTaskOfPipeline((issue.value as Issue).pipeline);
});

const allowApplyTaskTransition = (transition: TaskStatusTransition) => {
  if (transition.to === "PENDING") {
    // "Approve" is disabled when the task checks are not ready.
    const summary = taskCheckRunSummary(currentTask.value);
    if (summary.runningCount > 0 || summary.errorCount > 0) {
      return false;
    }
  }
  return true;
};

const getButtonActionList = (transition: TaskStatusTransition) => {
  const actionList: TaskStatusTransitionButtonAction[] = [];
  const { type, buttonName, buttonType } = transition;
  actionList.push({
    key: `${type}-TASK`,
    text: t(buttonName),
    type: buttonType,
    params: { transition, target: "TASK" },
  });

  if (allowApplyTaskTransitionToStage(transition)) {
    actionList.push({
      key: `${type}-STAGE`,
      text: t("issue.action-to-current-stage", {
        action: t(buttonName),
      }),
      type: buttonType,
      params: { transition, target: "STAGE" },
    });
  }

  return actionList;
};

const onClickTaskStatusTransitionActionButton = (
  action: TaskStatusTransitionButtonAction
) => {
  const { transition, target } = action.params;
  tryStartStageOrTaskStatusTransition(transition, target);
};

const allowIssueStatusTransition = (
  transition: IssueStatusTransition
): boolean => {
  if (transition.type == "RESOLVE") {
    const template = issueTemplate.value;
    // Returns false if any of the required output fields is not provided.
    for (let i = 0; i < template.outputFieldList.length; i++) {
      const field = template.outputFieldList[i];
      if (!field.resolved(issueContext.value)) {
        return false;
      }
    }
    return true;
  }
  return true;
};

const tryStartIssueStatusTransition = (transition: IssueStatusTransition) => {
  updateStatusModalState.mode = "ISSUE";
  updateStatusModalState.okText = t(transition.buttonName);
  switch (transition.type) {
    case "RESOLVE":
      updateStatusModalState.style = "SUCCESS";
      updateStatusModalState.title = t("issue.status-transition.modal.resolve");
      break;
    case "CANCEL":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = t("issue.status-transition.modal.cancel");
      break;
    case "REOPEN":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = t("issue.status-transition.modal.reopen");
      break;
  }
  updateStatusModalState.transition = transition;
  updateStatusModalState.show = true;
};

const doIssueStatusTransition = (
  transition: IssueStatusTransition,
  comment: string
) => {
  changeIssueStatus(transition.to, comment);
};

const allowCreate = computed(() => {
  const newIssue = issue.value as IssueCreate;

  if (isEmpty(newIssue.name)) {
    return false;
  }

  if (newIssue.assigneeId == UNKNOWN_ID) {
    return false;
  }

  if (IssueTypeWithStatement.includes(newIssue.type)) {
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

const onSubmit = async (comment: string) => {
  const cleanup = () => {
    updateStatusModalState.isTransiting = false;
    updateStatusModalState.show = false;
  };

  updateStatusModalState.isTransiting = true;
  // Trying to avoid some kind of concurrency and race condition, we fetch the
  // latest snapshot of issue from the server-side and check whether this
  // transition is applicable again.
  const latestIssue = await useIssueStore().fetchIssueById(
    (issue.value as Issue).id
  );

  if (updateStatusModalState.mode == "ISSUE") {
    const targetTransition =
      updateStatusModalState.transition as IssueStatusTransition;
    const applicableList = getApplicableIssueStatusTransitionList(latestIssue);
    if (!isApplicableTransition(targetTransition, applicableList)) {
      return cleanup();
    }
    doIssueStatusTransition(targetTransition, comment);
  } else if (updateStatusModalState.mode === "STAGE") {
    const targetTransition =
      updateStatusModalState.transition as StageStatusTransition;
    const applicableList = getApplicableStageStatusTransitionList(latestIssue);
    if (!isApplicableTransition(targetTransition, applicableList)) {
      return cleanup();
    }
    doStageStatusTransition(
      targetTransition,
      updateStatusModalState.payload as Stage,
      comment
    );
  } else if (updateStatusModalState.mode == "TASK") {
    const targetTransition =
      updateStatusModalState.transition as TaskStatusTransition;
    const applicableList = getApplicableTaskStatusTransitionList(latestIssue);
    if (!isApplicableTransition(targetTransition, applicableList)) {
      return cleanup();
    }
    doTaskStatusTransition(
      targetTransition,
      updateStatusModalState.payload as Task,
      comment
    );
  }

  cleanup();
};

const allowApplyTaskTransitionToStage = (transition: TaskStatusTransition) => {
  // Only available for the issue type of schema.update and data.update.
  const stage = currentTask.value.stage;

  // Only available when the stage has multiple tasks.
  if (stage.taskList.length <= 1) {
    return false;
  }

  // Available to apply a taskStatusTransition to the stage when the transition
  // type is also applicable to the stage.
  return (
    applicableStageStatusTransitionList.value.findIndex(
      (t) => t.type === transition.type
    ) >= 0
  );
};
</script>
