<template>
  <div class="flex flex-col gap-y-4">
    <!-- Execution History Section -->
    <div class="flex flex-col gap-y-3">
      <div class="flex items-center justify-between">
        <h3 class="text-base font-medium">
          {{ $t("task-run.history") }}
        </h3>
      </div>

      <!-- Task Runs Table -->
      <div v-if="allTaskRuns.length > 0">
        <TaskRunTable
          :task-runs="visibleTaskRuns"
          :show-database-column="true"
          :virtual-scroll="expanded && allTaskRuns.length > VIRTUAL_SCROLL_THRESHOLD"
          :max-height="expanded && allTaskRuns.length > VIRTUAL_SCROLL_THRESHOLD ? '80vh' : undefined"
        />
        <div v-if="hasMore" class="flex justify-center mt-2">
          <NButton quaternary size="small" @click="expanded = !expanded">
            <template v-if="expanded">
              {{ $t("common.collapse") }}
            </template>
            <template v-else>
              {{ $t("common.show-more") }}
              ({{ remainingCount }} {{ $t("common.remaining") }})
            </template>
          </NButton>
        </div>
      </div>

      <!-- No task runs message -->
      <div v-else class="text-center text-control-light py-8">
        {{ $t("common.no-data") }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import TaskRunTable from "@/components/RolloutV1/components/TaskRun/TaskRunTable.vue";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { extractTaskUID } from "@/utils";
import { usePlanContext } from "../../../logic";
import { useExpandableList, VIRTUAL_SCROLL_THRESHOLD } from ".";

const { plan, rollout, taskRuns } = usePlanContext();

// Get the export data spec
const exportDataSpec = computed(() => {
  return plan.value.specs.find(
    (spec) => spec.config?.case === "exportDataConfig"
  );
});

// Find export tasks related to this spec
const exportTasks = computed(() => {
  if (!rollout.value || !exportDataSpec.value) return [];

  const tasks = [];
  for (const stage of rollout.value.stages) {
    for (const task of stage.tasks) {
      if (task.specId === exportDataSpec.value.id) {
        tasks.push(task);
      }
    }
  }
  return tasks;
});

// Get all task runs for export tasks (empty if no rollout)
const allTaskRuns = computed((): TaskRun[] => {
  if (!rollout.value || exportTasks.value.length === 0) return [];

  const exportTaskUIDs = new Set(
    exportTasks.value.map((task) => extractTaskUID(task.name))
  );

  return taskRuns.value.filter((taskRun) =>
    exportTaskUIDs.has(extractTaskUID(taskRun.name))
  );
});

const {
  expanded,
  hasMore,
  visibleItems: visibleTaskRuns,
  remainingCount,
} = useExpandableList(allTaskRuns);
</script>
