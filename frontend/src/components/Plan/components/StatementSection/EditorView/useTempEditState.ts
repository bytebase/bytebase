import { computed, nextTick, ref, watch } from "vue";
import { UNKNOWN_ID } from "@/types";
import { usePlanContext } from "../../../logic";
import { useSpecSheet } from "../useSpecSheet";

export type EditState = {
  isEditing: boolean;
  statement: string;
};

export const useTempEditState = (state: EditState) => {
  const { isCreating, selectedSpec } = usePlanContext();
  const { sheet, sheetName, sheetReady, sheetStatement } = useSpecSheet();

  let stopWatching = () => {
    // noop
  };

  const startWatching = () => {
    const tempEditStateMap = new Map<string, EditState>();
    const isSwitchingSpec = ref(false);

    const selectedSpecId = computed((): string => {
      if (isCreating.value) return String(UNKNOWN_ID);
      return selectedSpec.value.id;
    });

    watch(selectedSpecId, () => {
      // When switching spec id, set the switch flag to true
      // to temporarily disable change listeners.
      isSwitchingSpec.value = true;
      nextTick(() => {
        isSwitchingSpec.value = false;
      });
    });

    const handleEditChange = () => {
      // When we are switching between specs, this will also be triggered.
      // But we shouldn't update the temp store.
      if (isSwitchingSpec.value) {
        return;
      }
      // Save the temp edit state before switching spec.
      tempEditStateMap.set(selectedSpecId.value, {
        isEditing: state.isEditing,
        statement: state.statement,
      });
    };

    const afterSpecIdChange = (id: string) => {
      // Try to restore the saved temp edit state after switching spec.
      const storedState = tempEditStateMap.get(id);
      if (storedState) {
        // If found the stored temp edit state, restore it.
        Object.assign(state, storedState);
      } else {
        // Restore to the spec's default state otherwise.
        state.isEditing = false;
        state.statement = sheetStatement.value;
      }
    };

    // Save the temp editing state before switching specs
    const stopWatchBeforeChange = watch(
      [() => state.isEditing, () => state.statement],
      handleEditChange,
      { immediate: true }
    );
    const stopWatchAfterChange = watch(
      selectedSpecId,
      afterSpecIdChange,
      { flush: "post" } // Listen to the event AFTER selectedSpecId changed
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
        // If we are opening an existed plan, we should listen and store the
        // temp editing states.
        stopWatching = startWatching();
      } else {
        // If we are creating an plan, we don't need the temp editing state
        // feature since all specs are still in editing mode.
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
