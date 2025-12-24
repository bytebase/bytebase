<template>
  <div class="flex flex-row gap-1 items-center">
    <template v-for="status in TASK_STATUS_FILTERS" :key="status">
      <NTag
        v-if="getTaskCount(status) > 0"
        round
        checkable
        :checked="selectedStatuses.includes(status)"
        @update:checked="
          (checked) => {
            handleStatusToggle(status, checked);
          }
        "
      >
        <template #avatar>
          <TaskStatus :status="status" size="small" />
        </template>
        <div class="flex flex-row items-center gap-1">
          <span class="select-none text-base">{{
            stringifyTaskStatus(status)
          }}</span>
          <span class="select-none text-base font-medium">{{
            getTaskCount(status)
          }}</span>
        </div>
      </NTag>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { TASK_STATUS_FILTERS } from "@/components/RolloutV1/constants/task";
import type {
  Stage,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus } from "@/utils";

const props = defineProps<{
  stage: Stage;
  selectedStatuses: Task_Status[];
}>();

const emit = defineEmits<{
  (event: "update:selected-statuses", statuses: Task_Status[]): void;
}>();

const allTasks = computed(() => {
  return props.stage.tasks;
});

const getTaskCount = (status: Task_Status) => {
  return allTasks.value.filter((task) => task.status === status).length;
};

const handleStatusToggle = (status: Task_Status, checked: boolean) => {
  const newStatuses = checked
    ? [...props.selectedStatuses, status]
    : props.selectedStatuses.filter((s) => s !== status);
  emit("update:selected-statuses", newStatuses);
};
</script>
