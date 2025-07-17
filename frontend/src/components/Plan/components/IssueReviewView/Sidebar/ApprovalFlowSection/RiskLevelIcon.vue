<template>
  <NTooltip v-if="riskLevel">
    <template #trigger>
      <ShieldCheckIcon
        v-if="riskLevel === Issue_RiskLevel.LOW"
        class="w-4 h-4 text-success"
      />
      <ShieldAlertIcon
        v-else-if="riskLevel === Issue_RiskLevel.MODERATE"
        class="w-4 h-4 text-warning"
      />
      <ShieldAlertIcon
        v-else-if="riskLevel === Issue_RiskLevel.HIGH"
        class="w-4 h-4 text-error"
      />
    </template>
    <span>{{ riskLevelText }}</span>
    <span class="opacity-60 ml-1">({{ $t("issue.risk-level.self") }})</span>
  </NTooltip>
</template>

<script setup lang="ts">
import { ShieldCheckIcon, ShieldAlertIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Issue_RiskLevel } from "@/types/proto-es/v1/issue_service_pb";

const props = defineProps<{
  riskLevel: Issue_RiskLevel;
}>();

const { t } = useI18n();

const riskLevelText = computed(() => {
  switch (props.riskLevel) {
    case Issue_RiskLevel.LOW:
      return t("issue.risk-level.low");
    case Issue_RiskLevel.MODERATE:
      return t("issue.risk-level.moderate");
    case Issue_RiskLevel.HIGH:
      return t("issue.risk-level.high");
    default:
      return "";
  }
});
</script>
