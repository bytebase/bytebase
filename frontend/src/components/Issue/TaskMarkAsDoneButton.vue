<template>
  <div
    v-if="allowMarkTaskAsDone"
    class="textinfolabel hover:text-accent cursor-pointer"
    v-bind="$attrs"
    @click="state.showModal = true"
  >
    {{ $t("task.mark-as-done") }}
  </div>

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
