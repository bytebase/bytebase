<template>
  <OptionRow v-if="shouldShowCombined">
    <template #label>
      {{ $t("plan.isolation-level.self") }}
    </template>
    <template #tooltip>
      {{ $t("plan.isolation-level.description") }}
    </template>
    <IsolationLevelSelect />
  </OptionRow>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import OptionRow from "../OptionRow.vue";
import { useTransactionModeSettingContext } from "../TransactionModeSection/context";
import { useIsolationLevelSettingContext } from "./context";
import IsolationLevelSelect from "./IsolationLevelSelect.vue";

// Check if isolation level section should be shown based on context
const { shouldShow } = useIsolationLevelSettingContext();

// Only show isolation level when transaction mode is ON and context allows it
const { transactionMode } = useTransactionModeSettingContext();

const shouldShowCombined = computed(() => {
  return shouldShow.value && transactionMode.value === "on";
});
</script>
