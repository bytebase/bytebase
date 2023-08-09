<template>
  <NDropdown
    v-if="allowSkipTask"
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
import { DropdownOption, NDropdown } from "naive-ui";
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { Issue, Task } from "@/types";
import { canSkipTask, TaskStatusTransition } from "@/utils";
import { useExtraIssueLogic, useIssueLogic } from "../logic";
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

const { create, issue } = useIssueLogic();
const { changeTaskStatus } = useExtraIssueLogic();

const allowSkipTask = computed(() => {
  if (create.value) {
    return false;
  }

  return canSkipTask(
    props.task,
    issue.value as Issue,
    true /* activeOnly */,
    false /* !failedOnly */
  );
});

const options = computed(() => {
  const list: DropdownOption[] = [];
  if (allowSkipTask.value) {
    list.push({
      key: "skip",
      label: t("task.skip"),
    });
  }
  return list;
});

const handleSelect = (key: string) => {
  if (key === "skip") {
    state.showModal = true;
  }
};

const confirmButtonText = computed(() => t("task.skip"));

const transition = computed((): TaskStatusTransition => {
  return {
    buttonType: "PRIMARY",
    buttonName: t("task.skip"),
    type: "SKIP",
    to: "DONE",
  };
});

const modalTitle = computed(() => {
  return t("task.skip-modal-title", {
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
