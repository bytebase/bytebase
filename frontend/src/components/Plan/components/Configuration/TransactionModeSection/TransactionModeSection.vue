<template>
  <div class="w-full flex items-center gap-3">
    <div class="flex items-center min-w-24">
      <label class="text-sm text-main">
        {{ $t("issue.transaction-mode.label") }}
      </label>
      <NTooltip>
        <template #trigger>
          <heroicons:information-circle
            class="w-4 h-4 text-control-light cursor-help"
          />
        </template>
        <template #default>
          <div class="max-w-xs space-y-1">
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
      </NTooltip>
    </div>
    <TransactionModeSwitch />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import TransactionModeSwitch from "./TransactionModeSwitch.vue";
import { useTransactionModeSettingContext } from "./context";

const { transactionMode } = useTransactionModeSettingContext();
</script>
