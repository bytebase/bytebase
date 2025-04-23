<template>
  <NTooltip v-if="shown" placement="bottom">
    <template #trigger>
      <NTag size="small" round strong>
        {{ riskLevelText }}
        <template #icon>
          <OctagonAlertIcon class="w-4 h-auto" />
        </template>
      </NTag>
    </template>
    <template #default>
      <div class="max-w-[22rem]">
        {{ $t("issue.risk-level.self") }}
      </div>
    </template>
  </NTooltip>
</template>

<script lang="ts" setup>
import { OctagonAlertIcon } from "lucide-vue-next";
import { NTooltip, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueContext } from "@/components/IssueV1";
import { Issue_RiskLevel } from "@/types/proto/api/v1alpha/issue_service";

const { t } = useI18n();
const { issue, isCreating } = useIssueContext();

const riskLevel = computed(() => issue.value.riskLevel);

const shown = computed(() => {
  return (
    !isCreating.value &&
    riskLevel.value !== Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED
  );
});

const riskLevelText = computed(() => {
  if (riskLevel.value === Issue_RiskLevel.LOW) {
    return t("issue.risk-level.low");
  } else if (riskLevel.value === Issue_RiskLevel.MODERATE) {
    return t("issue.risk-level.moderate");
  } else if (riskLevel.value === Issue_RiskLevel.HIGH) {
    return t("issue.risk-level.high");
  }
  return "";
});
</script>
