import { type Ref, ref, watch } from "vue";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { addToSet, deleteFromSet } from "../utils/reactivity";

export interface UseTaskCollapseReturn {
  expandedTaskIds: Ref<Set<string>>;
  isTaskExpanded: (task: Task) => boolean;
  toggleExpand: (task: Task) => void;
}

export const useTaskCollapse = (tasks: Ref<Task[]>): UseTaskCollapseReturn => {
  const expandedTaskIds = ref<Set<string>>(new Set());
  const lastTaskNames = ref<string>("");

  const isTaskExpanded = (task: Task): boolean => {
    return expandedTaskIds.value.has(task.name);
  };

  const toggleExpand = (task: Task) => {
    if (expandedTaskIds.value.has(task.name)) {
      deleteFromSet(expandedTaskIds, task.name);
    } else {
      addToSet(expandedTaskIds, task.name);
    }
  };

  // Auto-expand first non-completed task when tasks change (initial load or stage switch)
  watch(
    tasks,
    (newTasks) => {
      if (newTasks.length === 0) {
        return;
      }

      // Create a unique identifier for the current task set
      const currentTaskNames = newTasks.map((t) => t.name).join(",");

      // If tasks have changed (stage switch) or it's the first load, reset and auto-expand
      if (currentTaskNames !== lastTaskNames.value) {
        lastTaskNames.value = currentTaskNames;

        // Clear expanded tasks from previous stage
        expandedTaskIds.value.clear();

        // Auto-expand first non-completed task
        const firstActiveTask = newTasks.find(
          (task) =>
            task.status !== Task_Status.DONE &&
            task.status !== Task_Status.CANCELED
        );
        const taskToExpand = firstActiveTask || newTasks[0];
        addToSet(expandedTaskIds, taskToExpand.name);
      }
    },
    { immediate: true }
  );

  return {
    expandedTaskIds,
    isTaskExpanded,
    toggleExpand,
  };
};
