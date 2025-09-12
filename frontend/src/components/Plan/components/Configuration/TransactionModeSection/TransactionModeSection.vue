<template>
  <div
    class="w-full flex items-center justify-start gap-x-2 flex-wrap whitespace-nowrap"
  >
    <TransactionModeSwitch />
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          <span>{{ $t("issue.transaction-mode.label") }}</span>
        </div>
      </template>
      <template #default>
        <div class="max-w-[20rem]">
          <div v-if="transactionMode === 'on'">
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

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import TransactionModeSwitch from "./TransactionModeSwitch.vue";
import { useTransactionModeSettingContext } from "./context";

const { transactionMode } = useTransactionModeSettingContext();
</script>
