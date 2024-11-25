import { computed, nextTick, ref, watch } from "vue";
import { UNKNOWN_ID } from "@/types";
import { useIssueContext } from "../../../logic";
import { useTaskSheet } from "../useTaskSheet";

export type EditState = {
  isEditing: boolean;
  statement: string;
};

export const useTempEditState = (state: EditState) => {
  const { isCreating, selectedTask } = useIssueContext();
  const { sheet, sheetName, sheetReady, sheetStatement } = useTaskSheet();

  let stopWatching = () => {
    // noop
  };

  const startWatching = () => {
    const tempEditStateMap = new Map<string, EditState>();
    const isSwitchingTask = ref(false);

    // The issue page is polling the issue entity, making the reference obj
    // of `selectedTask` changes every time.
    // So we need to watch the id instead of the object ref.
    const selectedTaskName = computed((): string => {
      if (isCreating.value) return String(UNKNOWN_ID);
      return selectedTask.value.name;
    });

    watch(selectedTaskName, () => {
      // When switching task, set the switch flag to true
      // to temporarily disable change listeners.
      isSwitchingTask.value = true;
      nextTick(() => {
        isSwitchingTask.value = false;
      });
    });

    const handleEditChange = () => {
      // When we are switching between tasks, this will also be triggered.
      // But we shouldn't update the temp store.
      if (isSwitchingTask.value) {
        return;
      }
      // Save the temp edit state before switching task.
      tempEditStateMap.set(selectedTaskName.value, {
        isEditing: state.isEditing,
        statement: state.statement,
      });
    };

    const afterTaskNameChange = (name: string) => {
      // Try to restore the saved temp edit state after switching task.
      const storedState = tempEditStateMap.get(name);
      if (storedState) {
        // If found the stored temp edit state, restore it.
        Object.assign(state, storedState);
      } else {
        // Restore to the task's default state otherwise.
        state.isEditing = false;
        state.statement = sheetStatement.value;
      }
    };

    // Save the temp editing state before switching tasks
    const stopWatchBeforeChange = watch(
      [() => state.isEditing, () => state.statement],
      handleEditChange,
      { immediate: true }
    );
    const stopWatchAfterChange = watch(
      selectedTaskName,
      afterTaskNameChange,
      { flush: "post" } // Listen to the event AFTER selectedTaskId changed
    );

    return () => {
      tempEditStateMap.clear();
      stopWatchBeforeChange();
      stopWatchAfterChange();
    };
  };

  watch(
    isCreating,
    () => {
      if (!isCreating.value) {
        // If we are opening an existed issue, we should listen and store the
        // temp editing states.
        stopWatching = startWatching();
      } else {
        // If we are creating an issue, we don't need the temp editing state
        // feature since all tasks are still in editing mode.
        if (stopWatching) {
          stopWatching();
        }
      }
    },
    { immediate: true }
  );

  const reset = () => {
    stopWatching();

    if (!isCreating.value) {
      stopWatching = startWatching();
    }
  };

  return {
    reset,
    sheet,
    sheetName,
    sheetReady,
    sheetStatement,
  };
};
