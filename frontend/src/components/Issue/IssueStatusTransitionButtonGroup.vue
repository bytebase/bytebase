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
          preference-key="task-status-transition"
          data-label="bb-issue-status-transition-button"
          :action-list="getButtonActionList(transition)"
          @click="(action) => onClickTaskStatusTransitionActionButton(action as TaskStatusTransitionButtonAction)"
        />
      </template>
      <template v-if="applicableIssueStatusTransitionList.length > 0">
        <button
          id="user-menu"
          type="button"
          class="text-control-light"
          aria-label="User menu"
          aria-haspopup="true"
          @click.prevent="menu?.toggle($event, {})"
          @contextmenu.capture.prevent="menu?.toggle($event, {})"
        >
          <heroicons-solid:dots-vertical class="w-6 h-6" />
        </button>
        <BBContextMenu ref="menu" class="origin-top-right mt-10 w-42">
          <template
            v-for="(transition, index) in applicableIssueStatusTransitionList"
            :key="index"
          >
            <div
              class="menu-item"
              role="menuitem"
              @click.prevent="tryStartIssueStatusTransition(transition)"
            >
              {{ $t(transition.buttonName) }}
            </div>
          </template>
        </BBContextMenu>
      </template>
    </div>
    <template v-else>
      <div
        if="applicableIssueStatusTransitionList.length > 0"
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
</template>

<script lang="ts" setup>
import { computed, reactive, Ref, ref } from "vue";
import { isEmpty } from "lodash-es";
import { useI18n } from "vue-i18n";
import type { StageStatusTransition, TaskStatusTransition } from "@/utils";
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
import { BBContextMenu } from "@/bbkit";
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
import type { ButtonAction } from "@/bbkit/BBContextMenuButton.vue";

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

type TaskStatusTransitionButtonAction = ButtonAction<{
  transition: TaskStatusTransition;
  target: "TASK" | "STAGE";
}>;

const { t } = useI18n();
const menu = ref<InstanceType<typeof BBContextMenu>>();

const {
  create,
  issue,
  template: issueTemplate,
  activeTaskOfPipeline,
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

const tryStartStageOrTaskStatusTransition = (
  transition: TaskStatusTransition | StageStatusTransition,
  mode: "STAGE" | "TASK"
) => {
  updateStatusModalState.mode = mode;
  updateStatusModalState.okText = t(transition.buttonName);
  const task = currentTask.value;
  const payload = mode === "TASK" ? task : task.stage;
  const name = payload.name;
  switch (transition.type) {
    case "RUN":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.run")} '${name}'?`;
      break;
    case "APPROVE":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.approve")} '${name}'?`;
      break;
    case "RETRY":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.retry")} '${name}'?`;
      break;
    case "CANCEL":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.cancel")} '${name}'?`;
      break;
    case "SKIP":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.skip")} '${name}'?`;
      break;
  }
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
        if (isEmpty(task.statement)) {
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
