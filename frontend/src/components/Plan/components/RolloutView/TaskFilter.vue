<template>
  <div class="flex flex-row justify-start items-center gap-4">
    <div class="flex items-center justify-between">
      <h3 class="text-base">
        {{ $t("common.tasks") }}
      </h3>
    </div>
    <div class="flex flex-row gap-1 items-center">
      <template v-for="status in TASK_STATUS_FILTERS" :key="status">
        <NTag
          v-if="getTaskCount(status) > 0"
          round
          checkable
          :checked="taskStatusList.includes(status)"
          @update:checked="
            (checked) => {
              emit(
                'update:taskStatusList',
                checked
                  ? [...taskStatusList, status]
                  : taskStatusList.filter((s) => s !== status)
              );
            }
          "
        >
          <template #avatar>
            <TaskStatus :status="status" size="small" />
          </template>
          <div class="flex flex-row items-center gap-2">
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
  </div>
</template>

<script lang="ts" setup>
import { flatten } from "lodash-es";
import { NTag } from "naive-ui";
import { computed } from "vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import type {
  Rollout,
  Task_Status,
  Stage,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status as TaskStatusEnum } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus } from "@/utils";
import { useRolloutViewContext } from "./context";

const props = defineProps<{
  rollout: Rollout;
  taskStatusList: Task_Status[];
  stage?: Stage;
}>();

const emit = defineEmits<{
  (event: "update:taskStatusList", taskStatusList: Task_Status[]): void;
}>();

const { mergedStages } = useRolloutViewContext();

const TASK_STATUS_FILTERS: Task_Status[] = [
  TaskStatusEnum.DONE,
  TaskStatusEnum.RUNNING,
  TaskStatusEnum.PENDING,
  TaskStatusEnum.FAILED,
  TaskStatusEnum.CANCELED,
  TaskStatusEnum.NOT_STARTED,
  TaskStatusEnum.SKIPPED,
];

const allTasks = computed(() => {
  if (props.stage) {
    // If a specific stage is provided, use only its tasks
    return props.stage.tasks;
  }
  // Otherwise, use all tasks from all stages
  return flatten(mergedStages.value.map((stage) => stage.tasks));
});

const getTaskCount = (status: Task_Status) => {
  return allTasks.value.filter((task) => task.status === status).length;
};
</script>
