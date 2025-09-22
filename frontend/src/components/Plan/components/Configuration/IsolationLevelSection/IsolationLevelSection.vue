<template>
  <div v-if="shouldShowCombined" class="w-full flex items-center gap-3">
    <div class="flex items-center min-w-24">
      <label class="text-sm text-main">
        {{ $t("plan.isolation-level.self") }}
      </label>
      <NTooltip>
        <template #trigger>
          <heroicons:information-circle
            class="w-4 h-4 text-control-light cursor-help"
          />
        </template>
        <template #default>
          <div class="max-w-xs">
            {{ $t("plan.isolation-level.self") }}
          </div>
        </template>
      </NTooltip>
    </div>
    <IsolationLevelSelect />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useTransactionModeSettingContext } from "../TransactionModeSection/context";
import IsolationLevelSelect from "./IsolationLevelSelect.vue";
import { useIsolationLevelSettingContext } from "./context";

// Check if isolation level section should be shown based on context
const { shouldShow } = useIsolationLevelSettingContext();

// Only show isolation level when transaction mode is ON and context allows it
const { transactionMode } = useTransactionModeSettingContext();

const shouldShowCombined = computed(() => {
  return shouldShow.value && transactionMode.value === "on";
});
</script>
