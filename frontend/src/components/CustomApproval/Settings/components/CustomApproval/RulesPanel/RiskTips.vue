<template>
  <i18n-t
    v-if="showTips"
    tag="span"
    keypath="custom-approval.approval-flow.risk-not-configured-tips"
  >
    <template #link>
      <router-link to="/setting/risk-center" class="normal-link">
        {{ $t("custom-approval.approval-flow.the-related-risk-rules") }}
      </router-link>
    </template>
  </i18n-t>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRiskStore } from "@/store";
import { DEFAULT_RISK_LEVEL } from "@/types";
import type { Risk_Source } from "@/types/proto/v1/risk_service";

const props = defineProps<{
  level: number;
  source: Risk_Source;
  rule: string | undefined;
}>();

const riskStore = useRiskStore();

const showTips = computed(() => {
  const { level, source, rule } = props;
  if (!rule) return false;
  if (level === DEFAULT_RISK_LEVEL) return false;
  const riskList = riskStore.riskList.filter(
    (risk) => risk.active && risk.level === level && risk.source == source
  );
  // Show a tips to configure risk rules when
  // - the approval flow is configured (not "Skip approval")
  // - AND the number of corresponding risk rules is 0
  return riskList.length === 0;
});
</script>
