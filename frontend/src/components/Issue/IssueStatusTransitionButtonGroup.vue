<template>
  <template v-if="create">
    <button
      type="button"
      class="btn-primary px-4 py-2"
      :disabled="!allowCreate"
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
        <button
          type="button"
          :class="transition.buttonClass"
          @click.prevent="tryStartTaskStatusTransition(transition)"
        >
          {{ $t(transition.buttonName) }}
        </button>
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
import type { TaskStatusTransition } from "@/utils";
import type {
  Issue,
  IssueCreate,
  IssueStatusTransition,
  Principal,
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

export type IssueContext = {
  currentUser: Principal;
  create: boolean;
  issue: Issue | IssueCreate;
};

interface UpdateStatusModalState {
  mode: "ISSUE" | "TASK";
  show: boolean;
  style: string;
  okText: string;
  title: string;
  transition?: IssueStatusTransition | TaskStatusTransition;
  payload?: Task;
  isTransiting: boolean;
}

const { t } = useI18n();
const menu = ref<InstanceType<typeof BBContextMenu>>();

const {
  create,
  issue,
  template: issueTemplate,
  activeTaskOfPipeline,
  doCreate,
} = useIssueLogic();
const { changeIssueStatus, changeTaskStatus } = useExtraIssueLogic();

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
  getApplicableTaskStatusTransitionList,
} = useIssueTransitionLogic(issue as Ref<Issue>);

const tryStartTaskStatusTransition = (transition: TaskStatusTransition) => {
  updateStatusModalState.mode = "TASK";
  updateStatusModalState.okText = t(transition.buttonName);
  const task = currentTask.value;
  switch (transition.type) {
    case "RUN":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.run")} '${task.name}'?`;
      break;
    case "APPROVE":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.approve")} '${task.name}'?`;
      break;
    case "RETRY":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.retry")} '${task.name}'?`;
      break;
    case "CANCEL":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.cancel")} '${task.name}'?`;
      break;
    case "SKIP":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.skip")} '${task.name}'?`;
      break;
  }
  updateStatusModalState.transition = transition;
  updateStatusModalState.payload = task;
  updateStatusModalState.show = true;
};

const doTaskStatusTransition = (
  transition: TaskStatusTransition,
  task: Task,
  comment: string
) => {
  changeTaskStatus(task, transition.to, comment);
};

const currentTask = computed(() => {
  return activeTaskOfPipeline((issue.value as Issue).pipeline);
});

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
  } else if (updateStatusModalState.mode == "TASK") {
    const targetTransition =
      updateStatusModalState.transition as TaskStatusTransition;
    const applicableList = getApplicableTaskStatusTransitionList(latestIssue);
    if (!isApplicableTransition(targetTransition, applicableList)) {
      return cleanup();
    }
    doTaskStatusTransition(
      targetTransition,
      updateStatusModalState.payload!,
      comment
    );
  }

  cleanup();
};

const allowApplyTransitionToStage = (transition: TaskStatusTransition) => {
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
