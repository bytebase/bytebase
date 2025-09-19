<template>
  <div class="space-y-2">
    <!-- Tasks Display -->
    <div class="space-y-3">
      <!-- Task status summary with title -->
      <div
        v-if="exportTasks.length > 0"
        class="flex items-center justify-between"
      >
        <div class="flex items-center gap-2">
          <h3 class="text-base font-medium">
            {{ $t("common.task", exportTasks.length) }}
          </h3>
        </div>
      </div>

      <!-- Flattened task list -->
      <div v-if="exportTasks.length > 0" class="flex flex-wrap gap-2">
        <div
          v-for="task in exportTasks"
          :key="task.name"
          class="inline-flex items-center gap-2 px-2 py-1.5 border rounded transition-colors min-w-0 cursor-pointer"
          :class="{
            'bg-blue-50 border-blue-300': selectedTask?.name === task.name,
            'hover:bg-gray-50': selectedTask?.name !== task.name,
          }"
          @click="handleTaskClick(task)"
        >
          <!-- Task Status -->
          <div class="flex-shrink-0">
            <TaskStatus :status="task.status" size="tiny" />
          </div>

          <!-- Database Display -->
          <div class="flex-1 min-w-0">
            <DatabaseDisplay
              :database="task.target"
              :show-environment="true"
              size="medium"
            />
          </div>
        </div>
      </div>

      <!-- Task Runs for Selected Task -->
      <div
        v-if="selectedTask && selectedTaskRuns.length > 0"
        class="mt-4 space-y-2"
      >
        <div class="text-sm font-medium text-control">
          {{ $t("task-run.history") }}
        </div>
        <TaskRunTable :task="selectedTask" :task-runs="selectedTaskRuns" />
      </div>

      <!-- No tasks message -->
      <div
        v-else-if="exportTasks.length === 0"
        class="text-center text-control-light py-8"
      >
        {{ $t("common.no-data") }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watchEffect } from "vue";
import TaskRunTable from "@/components/Plan/components/RolloutView/TaskRunTable.vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import {
  Task_Status,
  type Task,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractTaskUID } from "@/utils";
import { usePlanContext } from "../../../logic";
import { usePlanContextWithRollout } from "../../../logic";

const { plan, rollout } = usePlanContext();
const { taskRuns } = usePlanContextWithRollout();

// Track selected task for showing task runs
const selectedTask = ref<Task | null>(null);

// Get task runs for the selected task
const selectedTaskRuns = computed((): TaskRun[] => {
  if (!selectedTask.value) return [];

  const taskUID = extractTaskUID(selectedTask.value.name);
  return taskRuns.value.filter(
    (taskRun) => extractTaskUID(taskRun.name) === taskUID
  );
});

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

// Auto-select the most relevant task
const autoSelectTask = () => {
  if (exportTasks.value.length === 0) {
    selectedTask.value = null;
    return;
  }

  // Priority order: Failed > Not started (Pending) > First task
  const failedTasks = exportTasks.value.filter(
    (task) => task.status === Task_Status.FAILED
  );
  const pendingTasks = exportTasks.value.filter(
    (task) =>
      task.status === Task_Status.PENDING ||
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.RUNNING
  );

  let taskToSelect: Task | null = null;

  if (failedTasks.length > 0) {
    // Select first failed task
    taskToSelect = failedTasks[0];
  } else if (pendingTasks.length > 0) {
    // Select first not started (pending) task
    taskToSelect = pendingTasks[0];
  } else {
    // Fallback to first task
    taskToSelect = exportTasks.value[0];
  }

  selectedTask.value = taskToSelect;
};

// Handle task selection
const handleTaskClick = (task: Task) => {
  if (selectedTask.value?.name === task.name) {
    selectedTask.value = null; // Deselect if clicking the same task
  } else {
    selectedTask.value = task; // Select the new task
  }
};

// Auto-select task when tasks change
watchEffect(() => {
  // Only auto-select if no task is currently selected
  if (!selectedTask.value && exportTasks.value.length > 0) {
    autoSelectTask();
  }
});
</script>
