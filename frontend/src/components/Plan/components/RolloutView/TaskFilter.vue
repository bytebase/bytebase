<template>
  <div class="flex flex-row justify-start items-center gap-4">
    <div class="flex items-center justify-between">
      <h3 class="text-base font-medium">
        {{ $t("common.tasks") }}
        <span>({{ totalTaskCount }})</span>
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
          <span class="select-none text-base">{{ getTaskCount(status) }}</span>
        </NTag>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { flatten } from "lodash-es";
import { NTag } from "naive-ui";
import { computed } from "vue";
import TaskStatus from "@/components/Rollout/RolloutDetail/Panels/kits/TaskStatus.vue";
import type { Rollout, Task_Status } from "@/types/proto/v1/rollout_service";
import { Task_Status as TaskStatusEnum } from "@/types/proto/v1/rollout_service";

const props = defineProps<{
  rollout: Rollout;
  taskStatusList: Task_Status[];
}>();

const emit = defineEmits<{
  (event: "update:taskStatusList", taskStatusList: Task_Status[]): void;
}>();

const TASK_STATUS_FILTERS: Task_Status[] = [
  TaskStatusEnum.NOT_STARTED,
  TaskStatusEnum.PENDING,
  TaskStatusEnum.RUNNING,
  TaskStatusEnum.DONE,
  TaskStatusEnum.FAILED,
  TaskStatusEnum.CANCELED,
  TaskStatusEnum.SKIPPED,
];

const allTasks = computed(() => {
  return flatten(props.rollout.stages.map((stage) => stage.tasks));
});

const totalTaskCount = computed(() => allTasks.value.length);

const getTaskCount = (status: Task_Status) => {
  return allTasks.value.filter((task) => task.status === status).length;
};
</script>
