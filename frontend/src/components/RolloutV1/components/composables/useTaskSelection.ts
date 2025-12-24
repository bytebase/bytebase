import { type ComputedRef, computed, type Ref, ref } from "vue";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  addToSet,
  clearSet,
  deleteFromSet,
  triggerSetReactivity,
} from "../utils/reactivity";
import { isTaskSelectable } from "../utils/taskStatus";

export interface UseTaskSelectionReturn {
  selectedTaskIds: Ref<Set<string>>;
  selectedTasks: ComputedRef<Task[]>;
  isTaskSelected: (task: Task) => boolean;
  isTaskSelectable: (task: Task) => boolean;
  toggleSelect: (task: Task) => void;
  selectAll: () => void;
  clearSelection: () => void;
}

export const useTaskSelection = (
  tasks: Ref<Task[]>
): UseTaskSelectionReturn => {
  const selectedTaskIds = ref<Set<string>>(new Set());

  const selectedTasks = computed(() => {
    return tasks.value.filter((task) => selectedTaskIds.value.has(task.name));
  });

  const isTaskSelected = (task: Task): boolean => {
    return selectedTaskIds.value.has(task.name);
  };

  const toggleSelect = (task: Task) => {
    if (!isTaskSelectable(task)) {
      return;
    }

    if (selectedTaskIds.value.has(task.name)) {
      deleteFromSet(selectedTaskIds, task.name);
    } else {
      addToSet(selectedTaskIds, task.name);
    }
  };

  const selectAll = () => {
    const selectableTasks = tasks.value.filter(isTaskSelectable);
    selectableTasks.forEach((task) => {
      selectedTaskIds.value.add(task.name);
    });
    triggerSetReactivity(selectedTaskIds);
  };

  const clearSelection = () => {
    clearSet(selectedTaskIds);
  };

  return {
    selectedTaskIds,
    selectedTasks,
    isTaskSelected,
    isTaskSelectable,
    toggleSelect,
    selectAll,
    clearSelection,
  };
};
