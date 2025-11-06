<template>
  <div class="flex items-center justify-between gap-x-2">
    <NInput
      v-if="!hideSearch"
      v-model:value="search"
      class="flex-1"
      :clearable="true"
      :placeholder="$t('custom-approval.risk-rule.search')"
    >
      <template #prefix>
        <heroicons:magnifying-glass class="w-4 h-4" />
      </template>
    </NInput>

    <slot name="suffix" />
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { useRiskFilter } from "./context";

defineProps<{
  hideSearch?: boolean;
}>();

const { search } = useRiskFilter();
</script>
