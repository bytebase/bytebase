<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <RiskLevelFilter />

      <div class="flex items-center justify-end gap-x-4">
        <NInput
          v-model:value="navigation.search"
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

    <div class="flex items-center justify-start">
      <RiskSourceFilter />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";

import RiskSourceFilter from "./RiskSourceFilter.vue";
import RiskLevelFilter from "./RiskLevelFilter.vue";
import { useRiskCenterContext } from "../context";

const { navigation } = useRiskCenterContext();
</script>
