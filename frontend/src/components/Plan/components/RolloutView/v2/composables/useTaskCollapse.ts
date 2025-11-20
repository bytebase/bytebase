import { type Ref, ref } from "vue";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { addToSet, deleteFromSet } from "../utils/reactivity";

export interface UseTaskCollapseReturn {
  expandedTaskIds: Ref<Set<string>>;
  isTaskExpanded: (task: Task) => boolean;
  toggleExpand: (task: Task) => void;
}

export const useTaskCollapse = (_tasks: Ref<Task[]>): UseTaskCollapseReturn => {
  const expandedTaskIds = ref<Set<string>>(new Set());

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

  return {
    expandedTaskIds,
    isTaskExpanded,
    toggleExpand,
  };
};
