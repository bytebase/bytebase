<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <div class="font-medium">
        {{ $t(titleKeyPath) }}
      </div>
    </div>
    <div>
      <RiskTable :risk-list="riskList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import RiskTable from "./RiskTable.vue";
import { Risk } from "@/types/proto/v1/risk_service";
import { PresetRiskLevelList } from "@/types";

const props = defineProps<{
  level: number;
  riskList: Risk[];
}>();

const titleKeyPath = computed(() => {
  const key = PresetRiskLevelList.find(
    (item) => item.level === props.level
  )?.name;
  if (!key) return "";
  return `custom-approval.security-rule.x-risk-rules.${key.toLowerCase()}`;
});
</script>
