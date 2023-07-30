<template>
  <BBModal
    :title="title"
    class="relative overflow-hidden"
    @close="$emit('cancel')"
  >
    <BatchTaskActionForm
      :issue="issue"
      :transition="transition"
      :task-list="taskList"
      :ok-text="okText"
      :title="title"
      @cancel="$emit('cancel')"
      @finish="$emit('updated')"
    />
  </BBModal>
</template>

<script setup lang="ts">
import { Ref, computed } from "vue";
import { useI18n } from "vue-i18n";

import { Issue, Task } from "@/types";
import { TaskStatusTransition } from "@/utils";
import BatchTaskActionForm from "./BatchTaskActionForm.vue";
import { useIssueLogic } from "../logic";

const props = defineProps<{
  transition: TaskStatusTransition;
  taskList: Task[];
}>();
defineEmits<{
  (event: "updated"): void;
  (event: "cancel"): void;
}>();

const { t } = useI18n();
const issueLogic = useIssueLogic();
const issue = issueLogic.issue as Ref<Issue>;

const okText = computed(() => {
  return t(props.transition.buttonName);
});

const title = computed(() => {
  return t("task.action-failed-in-current-stage", {
    action: okText.value,
  });
});
</script>
