<template>
  <span v-if="status">{{ status }}</span>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  TaskRunLogEntry_TaskRunStatusUpdate_Status,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const { t } = useI18n();

const status = computed(() => {
  const { type, taskRunStatusUpdate } = props.entry;
  if (
    type === TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE &&
    taskRunStatusUpdate
  ) {
    const { status } = taskRunStatusUpdate;
    if (status === TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_WAITING) {
      return t("issue.task-run.task-run-log.status-update.waiting");
    }
    if (status === TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_RUNNING) {
      return t("issue.task-run.task-run-log.status-update.running");
    }
  }
  return undefined;
});
</script>
