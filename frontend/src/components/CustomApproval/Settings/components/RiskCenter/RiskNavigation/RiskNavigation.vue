<template>
  <div class="flex justify-between items-center">
    <div class="flex items-center justify-start">
      <RiskSourceFilter />
    </div>

    <div class="flex items-center justify-end gap-x-4">
      <RiskLevelFilter />

      <NInput
        v-model:value="navigation.search"
        :placeholder="$t('custom-approval.security-rule.search')"
      >
        <template #prefix>
          <heroicons:magnifying-glass class="w-4 h-4" />
        </template>
      </NInput>
    </div>
  </div>
</template>

<script lang="ts" setup>
import RiskSourceFilter from "./RiskSourceFilter.vue";
import RiskLevelFilter from "./RiskLevelFilter.vue";
import { useRiskCenterContext } from "../context";

const { navigation } = useRiskCenterContext();
</script>
