<template>
  <div class="flex flex-row justify-start items-center gap-2">
    <span class="shrink-0 text-sm">{{
      $t("issue.transaction-mode.label")
    }}</span>
    <NTooltip>
      <template #trigger>
        <NSwitch
          v-model:value="isTransactionOn"
          size="small"
          :disabled="shouldDisableTransactionMode"
        >
          <template #checked>
            {{ $t("issue.transaction-mode.on") }}
          </template>
          <template #unchecked>
            {{ $t("issue.transaction-mode.off") }}
          </template>
        </NSwitch>
      </template>
      <template #default>
        <div class="max-w-xs">
          <div v-if="isTransactionOn">
            {{ $t("issue.transaction-mode.on-tooltip") }}
          </div>
          <div v-else>
            {{ $t("issue.transaction-mode.off-tooltip") }}
          </div>
        </div>
      </template>
    </NTooltip>
  </div>
</template>

<script setup lang="tsx">
import { NSwitch, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { getDefaultTransactionMode } from "@/utils";
import { useEditorContext } from "./context";
import { parseStatement, updateTransactionMode } from "./directiveUtils";

const editorContext = useEditorContext();
const { selectedTask } = useIssueContext();

// Transaction mode can be configured for all supported engines
const shouldDisableTransactionMode = computed(() => {
  return false;
});

// Default transaction mode
const getDefaultForCurrentTask = () => {
  return getDefaultTransactionMode();
};

const isTransactionOn = ref(getDefaultForCurrentTask());

// Watch for task changes to reset defaults
watch(
  () => selectedTask.value.name,
  () => {
    // Parse existing statement for transaction mode
    const parsed = parseStatement(editorContext.statement.value);
    if (parsed.transactionMode !== undefined) {
      isTransactionOn.value = parsed.transactionMode === "on";
    } else {
      // Use default if no directive found
      isTransactionOn.value = getDefaultForCurrentTask();
    }
  },
  { immediate: true }
);

// Update statement when transaction mode changes
watch(
  () => isTransactionOn.value,
  () => {
    updateStatementWithTransactionMode();
  }
);

const updateStatementWithTransactionMode = () => {
  const mode = isTransactionOn.value ? "on" : "off";
  const updatedStatement = updateTransactionMode(
    editorContext.statement.value,
    mode
  );
  editorContext.setStatement(updatedStatement);
};
</script>
