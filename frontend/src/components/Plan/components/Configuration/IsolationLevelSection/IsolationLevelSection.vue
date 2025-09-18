<template>
  <div
    v-if="shouldShow"
    class="w-full flex flex-row justify-start items-center gap-4"
  >
    <label class="w-28 shrink-0 text-sm text-control-light">
      {{ $t("plan.isolation-level.self") }}
    </label>
    <IsolationLevelSelect />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useTransactionModeSettingContext } from "../TransactionModeSection/context";
import IsolationLevelSelect from "./IsolationLevelSelect.vue";

// Only show isolation level when transaction mode is ON
const { transactionMode } = useTransactionModeSettingContext();

const shouldShow = computed(() => {
  return transactionMode.value === "on";
});
</script>
