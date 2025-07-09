<template>
  <div class="w-full flex flex-col mt-4">
    <!-- Task Operations Bar -->
    <TaskOperations
      :tasks="selectedTasks"
      :rollout="rollout"
      :stage="props.stage"
      :readonly="props.readonly"
      @refresh="handleRefresh"
      @action-confirmed="handleTaskActionConfirmed"
    />

    <!-- Task Table -->
    <div class="flex-1 overflow-hidden">
      <TaskTable
        :task-status-filter="props.taskStatusFilter"
        :selected-tasks="selectedTasks"
        :stage="props.stage"
        @update:selected-tasks="handleSelectedTasksUpdate"
        @refresh="handleRefresh"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import type {
  Task,
  Task_Status,
  Stage,
} from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../logic";
import TaskOperations from "./TaskOperations.vue";
import TaskTable from "./TaskTable.vue";

const props = defineProps<{
  taskStatusFilter: Task_Status[];
  stage: Stage;
  readonly?: boolean;
}>();

const { rollout, events } = usePlanContextWithRollout();
const selectedTasks = ref<Task[]>([]);

const handleSelectedTasksUpdate = (tasks: Task[]) => {
  selectedTasks.value = tasks;
};

const handleRefresh = () => {
  events.emit("status-changed", { eager: true });
};

const handleTaskActionConfirmed = () => {
  // Clear selection after action is confirmed.
  selectedTasks.value = [];
  handleRefresh();
};
</script>
