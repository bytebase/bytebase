<template>
  <NSwitch
    v-model:value="isTransactionOn"
    size="small"
    :disabled="!allowChange"
  />
</template>

<script setup lang="tsx">
import { NSwitch } from "naive-ui";
import { computed, watch } from "vue";
import { getDefaultTransactionMode } from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";
import {
  parseStatement,
  updateTransactionMode,
} from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import { useTransactionModeSettingContext } from "./context";

const { allowChange, transactionMode, events } =
  useTransactionModeSettingContext();
const selectedSpec = useSelectedSpec();
const { sheetStatement, updateSheetStatement } = useSpecSheet(selectedSpec);

// Default transaction mode
const getDefaultForCurrentTask = () => {
  return getDefaultTransactionMode();
};

const isTransactionOn = computed({
  get: () => transactionMode.value === "on",
  set: (value: boolean) => {
    transactionMode.value = value ? "on" : "off";
    updateStatementWithTransactionMode();
  },
});

// Initialize transaction mode from statement
const initializeFromStatement = () => {
  const parsed = parseStatement(sheetStatement.value);
  if (parsed.transactionMode !== undefined) {
    transactionMode.value = parsed.transactionMode;
  } else {
    transactionMode.value = getDefaultForCurrentTask() ? "on" : "off";
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

// Update statement when transaction mode changes
const updateStatementWithTransactionMode = () => {
  const mode = transactionMode.value;
  const updatedStatement = updateTransactionMode(sheetStatement.value, mode);
  updateSheetStatement(updatedStatement);
  events.emit("update");
};
</script>
