<template>
  <div class="space-y-2">
    <div class="flex items-center justify-start">
      <div class="font-medium text-base">
        {{ sourceText(source) }}
      </div>
      <div class="ml-1">({{ riskList.length }})</div>
    </div>
    <div>
      <RiskTable :risk-list="riskList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";
import { sourceText } from "../common";
import RiskTable from "./RiskTable.vue";

defineProps<{
  source: Risk_Source;
  riskList: Risk[];
}>();
</script>
