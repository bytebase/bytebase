<template>
  <NTooltip :disabled="!disabledTooltip" placement="bottom-end">
    <template #trigger>
      <NButton
        size="medium"
        tag="div"
        v-bind="actionButtonProps(action)"
        :disabled="disabled"
        @click="$emit('perform-action', action)"
      >
        <template v-if="action === 'EXPORT_DOWNLOAD'" #icon>
          <DownloadIcon class="w-5 h-5" />
        </template>
        {{ actionDisplayName(action) }}
      </NButton>
    </template>
    <template #default>
      {{ disabledTooltip }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { DownloadIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { usePlanContext } from "../../../../logic";
import type { UnifiedAction } from "./types";

defineProps<{
  action: UnifiedAction;
  disabled?: boolean;
  disabledTooltip?: string;
}>();

defineEmits<{
  (event: "perform-action", action: UnifiedAction): void;
}>();

const { t } = useI18n();
const { plan } = usePlanContext();

// Check if this is a database export plan
const isExportPlan = computed(() => {
  return plan.value.specs.some(
    (spec) => spec.config?.case === "exportDataConfig"
  );
});

const actionDisplayName = (action: UnifiedAction): string => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
      return t("common.approve");
    case "ISSUE_REVIEW_REJECT":
      return t("custom-approval.issue-review.send-back");
    case "ISSUE_REVIEW_RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review");
    case "ISSUE_STATUS_CLOSE":
      return t("issue.batch-transition.close");
    case "ISSUE_STATUS_REOPEN":
      return t("issue.batch-transition.reopen");
    case "ISSUE_STATUS_RESOLVE":
      return t("issue.batch-transition.resolve");
    case "ISSUE_CREATE":
      return t("plan.ready-for-review");
    case "PLAN_CLOSE":
      return t("common.close");
    case "PLAN_REOPEN":
      return t("common.reopen");
    case "ROLLOUT_START":
      return isExportPlan.value ? t("common.export") : t("common.rollout");
    case "ROLLOUT_CANCEL":
      return t("common.cancel");
    case "EXPORT_DOWNLOAD":
      return t("common.download");
  }
};

const actionButtonProps = (action: UnifiedAction) => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
    case "ISSUE_CREATE":
    case "ROLLOUT_START":
    case "EXPORT_DOWNLOAD":
      return {
        type: "primary" as const,
      };
    case "ISSUE_STATUS_RESOLVE":
      return {
        type: "success" as const,
      };
    case "ISSUE_REVIEW_RE_REQUEST":
    case "ISSUE_STATUS_REOPEN":
    case "ISSUE_REVIEW_REJECT":
    case "ISSUE_STATUS_CLOSE":
    case "PLAN_CLOSE":
    case "PLAN_REOPEN":
    case "ROLLOUT_CANCEL":
      return {
        type: "default" as const,
      };
  }
};
</script>
