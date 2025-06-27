<template>
  <div
    class="!w-80 bg-white z-[1] rounded-lg p-1"
    :class="
      twMerge(
        isCreated ? 'bg-white shadow' : 'bg-zinc-50 border-2 border-dashed'
      )
    "
  >
    <div
      class="w-full flex flex-row justify-between items-center gap-2 px-2 pt-2 pb-1"
    >
      <p class="text-base font-medium">
        {{ environmentStore.getEnvironmentByName(stage.environment).title }}
        <NTag v-if="!isCreated" round size="tiny">Preview</NTag>
      </p>
      <div v-if="isCreated">
        <RunTasksButton :stage="stage" @run-tasks="showRunTasksPanel = true" />
      </div>
    </div>
    <NVirtualList
      style="max-height: 80vh"
      :items="filteredTasks"
      :item-size="56"
    >
      <template #default="{ item: task }: { item: Task }">
        <div
          :key="task.name"
          class="w-full border-t border-zinc-50 flex items-center justify-start truncate px-2 py-3 min-h-[56px]"
        >
          <TaskStatus :status="task.status" size="small" class="shrink-0" />
          <TaskDatabaseName :task="task" class="ml-2 flex-1" />
          <div class="ml-auto flex items-center space-x-1 shrink-0">
            <NTooltip>
              <template #trigger>
                <NTag round size="tiny">{{ semanticTaskType(task.type) }}</NTag>
              </template>
              {{ $t("common.type") }}
            </NTooltip>

            <NTooltip v-if="extractSchemaVersionFromTask(task)">
              <template #trigger>
                <NTag round size="tiny">
                  {{ extractSchemaVersionFromTask(task) }}
                </NTag>
              </template>
              {{ $t("common.version") }}
            </NTooltip>
          </div>
        </div>
      </template>
    </NVirtualList>

    <!-- Task Rollout Action Panel -->
    <TaskRolloutActionPanel
      :action="showRunTasksPanel ? 'RUN_TASKS' : undefined"
      :stage="stage"
      @close="showRunTasksPanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import { NTag, NTooltip, NVirtualList } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { semanticTaskType } from "@/components/IssueV1";
import TaskStatus from "@/components/Rollout/RolloutDetail/Panels/kits/TaskStatus.vue";
import { useEnvironmentV1Store } from "@/store";
import {
  Stage,
  type Task,
  type Task_Status,
} from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";
import RunTasksButton from "./RunTasksButton.vue";
import TaskDatabaseName from "./TaskDatabaseName.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import { useRolloutViewContext } from "./context";

const props = defineProps<{
  stage: Stage;
  taskStatusFilter: Task_Status[];
}>();

const { t: $t } = useI18n();
const environmentStore = useEnvironmentV1Store();
const { rollout } = useRolloutViewContext();

const showRunTasksPanel = ref(false);

const isCreated = computed(() => {
  if (!rollout.value) {
    return false;
  }
  return rollout.value.stages.some(
    (stage) => stage.environment === props.stage.environment
  );
});

const filteredTasks = computed(() => {
  if (props.taskStatusFilter.length === 0) {
    return props.stage.tasks;
  }
  return props.stage.tasks.filter((task) =>
    props.taskStatusFilter.includes(task.status)
  );
});
</script>
