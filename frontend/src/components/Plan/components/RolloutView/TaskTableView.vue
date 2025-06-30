<template>
  <div class="w-full flex flex-col mt-4">
    <!-- Task Operations Bar -->
    <TaskOperations
      :tasks="selectedTasks"
      :rollout="rollout"
      @refresh="handleRefresh"
      @task-action-completed="handleTaskActionCompleted"
    />

    <!-- Task Table -->
    <div class="flex-1 overflow-hidden">
      <TaskTable
        :task-status-filter="taskStatusFilter"
        :selected-tasks="selectedTasks"
        @update:selected-tasks="handleSelectedTasksUpdate"
        @refresh="handleRefresh"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import type { Task, Task_Status } from "@/types/proto/v1/rollout_service";
import { usePlanContextWithRollout } from "../../logic";
import TaskOperations from "./TaskOperations.vue";
import TaskTable from "./TaskTable.vue";

defineProps<{
  taskStatusFilter: Task_Status[];
}>();

const { rollout, events } = usePlanContextWithRollout();
const selectedTasks = ref<Task[]>([]);

const handleSelectedTasksUpdate = (tasks: Task[]) => {
  selectedTasks.value = tasks;
};

const handleRefresh = () => {
  events.emit("status-changed", { eager: true });
};

const handleTaskActionCompleted = () => {
  // Clear selection after action is completed
  selectedTasks.value = [];
  handleRefresh();
};
</script>
