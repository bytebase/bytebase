<template>
  <NDropdown
    v-if="allowMarkTaskAsDone"
    trigger="click"
    placement="bottom-end"
    :options="options"
    @select="handleSelect"
  >
    <button
      id="user-menu"
      type="button"
      class="text-control-light p-0.5 rounded hover:bg-control-bg-hover"
      aria-label="User menu"
      aria-haspopup="true"
      v-bind="$attrs"
    >
      <heroicons-solid:dots-vertical class="w-4 h-4" />
    </button>
  </NDropdown>

  <BBModal
    v-if="state.showModal"
    :title="modalTitle"
    class="relative overflow-hidden"
    @close="state.showModal = false"
  >
    <div
      v-if="state.isLoading"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
    <StatusTransitionForm
      mode="TASK"
      :ok-text="confirmButtonText"
      :issue="(issue as Issue)"
      :task="task"
      :transition="transition!"
      :output-field-list="[]"
      @submit="onSubmit"
      @cancel="state.showModal = false"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { DropdownOption, NDropdown } from "naive-ui";

import type { Issue, Task, TaskStatus } from "@/types";
import { useCurrentUser } from "@/store";
import { useExtraIssueLogic, useIssueLogic } from "./logic";
import {
  activeTask,
  hasWorkspacePermission,
  TaskStatusTransition,
} from "@/utils";
import StatusTransitionForm from "./StatusTransitionForm.vue";

type LocalState = {
  showModal: boolean;
  isLoading: boolean;
};

const props = defineProps({
  task: {
    type: Object as PropType<Task>,
    required: true,
  },
});

const state = reactive<LocalState>({
  showModal: false,
  isLoading: false,
});

const { t } = useI18n();
const currentUser = useCurrentUser();

const { create, issue } = useIssueLogic();
const { changeTaskStatus } = useExtraIssueLogic();

const allowMarkTaskAsDone = computed(() => {
  if (create.value) {
    return false;
  }

  const { task } = props;

  const pipeline = (issue.value as Issue).pipeline;
  const isActiveTask = task.id === activeTask(pipeline).id;
  if (!isActiveTask) {
    return false;
  }

  const applicableStatusList: TaskStatus[] = ["PENDING_APPROVAL", "FAILED"];

  if (!applicableStatusList.includes(task.status)) {
    return false;
  }

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-issue",
      currentUser.value.role
    )
  ) {
    return true;
  }

  if (currentUser.value.id === (issue.value as Issue).assignee.id) {
    return true;
  }

  return false;
});

const options = computed(() => {
  const list: DropdownOption[] = [];
  if (allowMarkTaskAsDone.value) {
    list.push({
      key: "mark-task-as-done",
      label: t("task.mark-as-done"),
    });
  }
  return list;
});

const handleSelect = (key: string) => {
  if (key === "mark-task-as-done") {
    state.showModal = true;
  }
};

const confirmButtonText = computed(() => t("task.mark-as-done"));

const transition = computed((): TaskStatusTransition => {
  return {
    buttonType: "PRIMARY",
    buttonName: t("task.mark-as-done"),
    type: "SKIP",
    to: "DONE",
  };
});

const modalTitle = computed(() => {
  return t("task.mark-as-done-modal-title", {
    name: props.task.name,
  });
});

const onSubmit = async (comment: string) => {
  state.isLoading = true;
  try {
    changeTaskStatus(props.task, "DONE", comment);
  } finally {
    state.isLoading = false;
    state.showModal = false;
  }
};
</script>
