<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <div>
        <LevelFilter v-if="!hideLevelFilter" />
      </div>

      <div class="flex items-center justify-end gap-x-4">
        <NInput
          v-if="!hideSearch"
          v-model:value="search"
          :clearable="true"
          :placeholder="$t('custom-approval.security-rule.search')"
        >
          <template #prefix>
            <heroicons:magnifying-glass class="w-4 h-4" />
          </template>
        </NInput>

        <slot name="suffix" />
      </div>
    </div>

    <hr v-if="!hideSourceFilter" />

    <div v-if="!hideSourceFilter" class="flex items-center justify-start">
      <SourceFilter />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import LevelFilter from "./LevelFilter.vue";
import SourceFilter from "./SourceFilter.vue";
import { useRiskFilter } from "./context";

defineProps<{
  hideLevelFilter?: boolean;
  hideSourceFilter?: boolean;
  hideSearch?: boolean;
}>();

const { search } = useRiskFilter();
</script>
