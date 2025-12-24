<template>
  <div class="w-full">
    <TaskToolbar
      v-if="!readonly"
      :selected-tasks="selectedTasks"
      :all-tasks="filteredTasks"
      :is-task-selectable="isTaskSelectable"
      :stage="stage"
      @select-all="selectAll"
      @clear-selection="clearSelection"
      @action-complete="handleActionComplete"
    />

    <div class="task-list px-4 py-3 space-y-3">
      <TaskItem
        v-for="task in visibleTasks"
        :key="task.name"
        :task="task"
        :stage="stage"
        :is-expanded="isTaskExpanded(task)"
        :is-selected="isTaskSelected(task)"
        :is-selectable="!readonly && isTaskSelectable(task)"
        :in-select-mode="!readonly && selectedTasks.length > 0"
        :readonly="readonly"
        @toggle-expand="toggleExpand(task)"
        @toggle-select="toggleSelect(task)"
      />

      <div v-if="filteredTasks.length === 0" class="text-center py-8 text-gray-500">
        {{ $t("rollout.task.no-tasks") }}
      </div>

      <!-- Show More button -->
      <div
        v-if="hasMoreTasks"
        class="flex justify-center"
      >
        <button
          class="px-4 py-2 text-sm text-blue-600 hover:text-blue-700 hover:bg-blue-50 rounded-md transition-colors"
          @click="loadMoreTasks"
        >
          {{ $t("common.show-more") }} ({{ remainingTasksCount }} {{ $t("common.remaining") }})
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { useTaskCollapse } from "./composables/useTaskCollapse";
import { useTaskSelection } from "./composables/useTaskSelection";
import { DEFAULT_PAGE_SIZE } from "./constants";
import TaskItem from "./TaskItem.vue";
import TaskToolbar from "./TaskToolbar.vue";
import { compareTasksByStatus } from "./utils/taskStatus";

const props = withDefaults(
  defineProps<{
    stage: Stage;
    rollout: Rollout;
    filterStatuses: Task_Status[];
    readonly?: boolean;
  }>(),
  {
    readonly: false,
  }
);

const { events } = usePlanContextWithRollout();

const displayedTaskCount = ref(DEFAULT_PAGE_SIZE);

const filteredTasks = computed(() => {
  const tasks = props.stage.tasks;
  let result = tasks;

  if (props.filterStatuses.length > 0) {
    result = tasks.filter((task) => props.filterStatuses.includes(task.status));
  }

  // Sort by status priority (order defined in TASK_STATUS_FILTERS)
  return [...result].sort(compareTasksByStatus);
});

const visibleTasks = computed(() => {
  return filteredTasks.value.slice(0, displayedTaskCount.value);
});

const hasMoreTasks = computed(() => {
  return filteredTasks.value.length > displayedTaskCount.value;
});

const remainingTasksCount = computed(() => {
  return filteredTasks.value.length - displayedTaskCount.value;
});

const loadMoreTasks = () => {
  displayedTaskCount.value = Math.min(
    displayedTaskCount.value + DEFAULT_PAGE_SIZE,
    filteredTasks.value.length
  );
};

const { isTaskExpanded, toggleExpand } = useTaskCollapse(filteredTasks);

const {
  selectedTasks,
  isTaskSelected,
  isTaskSelectable,
  toggleSelect,
  selectAll,
  clearSelection,
} = useTaskSelection(filteredTasks);

const handleActionComplete = () => {
  clearSelection();
  // Trigger immediate refresh of rollout data after task actions
  events.emit("status-changed", { eager: true });
};

// Reset pagination when filters change, but preserve state on refresh
watch(
  () => props.filterStatuses,
  () => {
    displayedTaskCount.value = DEFAULT_PAGE_SIZE;
  }
);

// Ensure displayedTaskCount doesn't exceed available tasks
watch(
  () => filteredTasks.value.length,
  (newLength) => {
    if (displayedTaskCount.value > newLength) {
      displayedTaskCount.value = Math.max(DEFAULT_PAGE_SIZE, newLength);
    }
  }
);
</script>
