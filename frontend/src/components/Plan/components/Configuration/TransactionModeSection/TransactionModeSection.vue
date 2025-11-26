<template>
  <OptionRow>
    <template #label>
      {{ $t("issue.transaction-mode.label") }}
    </template>
    <template #tooltip>
      <div class="flex flex-col gap-y-1">
        <div v-if="transactionMode === 'on'">
          {{ $t("issue.transaction-mode.on-tooltip") }}
        </div>
        <div v-else>
          {{ $t("issue.transaction-mode.off-tooltip") }}
        </div>
        <div class="text-sm text-gray-400">
          {{ $t("issue.transaction-mode.execution-order-note") }}
        </div>
      </div>
    </template>
    <TransactionModeSwitch />
  </OptionRow>
</template>

<script lang="ts" setup>
import OptionRow from "../OptionRow.vue";
import { useTransactionModeSettingContext } from "./context";
import TransactionModeSwitch from "./TransactionModeSwitch.vue";

const { transactionMode } = useTransactionModeSettingContext();
</script>
