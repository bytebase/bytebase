<template>
  <NSwitch
    v-model:value="isTransactionOn"
    size="small"
    :disabled="!allowChange"
  />
</template>

<script setup lang="tsx">
import { NSwitch } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import {
  updateSpecSheetWithStatement,
  usePlanContext,
} from "@/components/Plan/logic";
import { getDefaultTransactionMode, setSheetStatement } from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";
import {
  parseStatement,
  updateTransactionMode,
} from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import { useTransactionModeSettingContext } from "./context";

const { allowChange, transactionMode, events } =
  useTransactionModeSettingContext();
const { selectedSpec } = useSelectedSpec();
const { sheetStatement, sheet, sheetReady } = useSpecSheet(selectedSpec);
const { plan, isCreating } = usePlanContext();

// Flag to prevent circular updates
const isUpdatingFromUI = ref(false);

const isTransactionOn = computed({
  get: () => transactionMode.value === "on",
  set: (value: boolean) => {
    isUpdatingFromUI.value = true;
    transactionMode.value = value ? "on" : "off";
    updateStatementWithTransactionMode();
    // Reset the flag after Vue's next tick to allow statement updates to propagate
    nextTick(() => {
      isUpdatingFromUI.value = false;
    });
  },
});

// Initialize transaction mode from statement
const initializeFromStatement = () => {
  if (!sheetReady.value || !sheetStatement.value) {
    // If sheet is not ready or statement is empty, use default
    transactionMode.value = getDefaultTransactionMode() ? "on" : "off";
    return;
  }

  const parsed = parseStatement(sheetStatement.value);
  if (parsed.transactionMode !== undefined) {
    transactionMode.value = parsed.transactionMode;
  } else {
    transactionMode.value = getDefaultTransactionMode() ? "on" : "off";
  }
};

// Watch for spec changes to reset defaults
watch(
  () => selectedSpec.value?.id,
  () => {
    initializeFromStatement();
  },
  { immediate: true }
);

// Watch for sheet statement changes to update transaction mode
// Only update when statement changes externally, not when we change it ourselves
watch(
  [sheetStatement, sheetReady],
  () => {
    if (!isUpdatingFromUI.value) {
      initializeFromStatement();
    }
  },
  { immediate: true }
);

// Update statement when transaction mode changes
const updateStatementWithTransactionMode = async () => {
  const mode = transactionMode.value;
  const updatedStatement = updateTransactionMode(sheetStatement.value, mode);

  if (isCreating.value) {
    // When creating a plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, updatedStatement);
  } else {
    // For created plans, create new sheet and update plan/spec
    await updateSpecSheetWithStatement(
      plan.value,
      selectedSpec.value,
      updatedStatement
    );
  }
  events.emit("update");
};
</script>
