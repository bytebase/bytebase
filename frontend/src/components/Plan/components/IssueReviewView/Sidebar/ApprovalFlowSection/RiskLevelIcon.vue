<template>
  <NTooltip v-if="riskLevel">
    <template #trigger>
      <ShieldCheckIcon
        v-if="riskLevel === RiskLevel.LOW"
        class="w-4 h-4 text-success"
      />
      <ShieldAlertIcon
        v-else-if="riskLevel === RiskLevel.MODERATE"
        class="w-4 h-4 text-warning"
      />
      <ShieldAlertIcon
        v-else-if="riskLevel === RiskLevel.HIGH"
        class="w-4 h-4 text-error"
      />
    </template>
    <span>{{ riskLevelText }}</span>
    <span class="opacity-60 ml-1">({{ $t("issue.risk-level.self") }})</span>
  </NTooltip>
</template>

<script setup lang="ts">
import { ShieldAlertIcon, ShieldCheckIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  riskLevel: RiskLevel;
}>();

const { t } = useI18n();

const riskLevelText = computed(() => {
  switch (props.riskLevel) {
    case RiskLevel.LOW:
      return t("issue.risk-level.low");
    case RiskLevel.MODERATE:
      return t("issue.risk-level.moderate");
    case RiskLevel.HIGH:
      return t("issue.risk-level.high");
    default:
      return "";
  }
});
</script>
