<template>
  <NTooltip :disabled="!disabledTooltip" placement="top">
    <template #trigger>
      <NButton
        size="medium"
        tag="div"
        v-bind="actionButtonProps(action)"
        :disabled="disabled"
        @click="$emit('perform-action', action)"
      >
        {{ actionDisplayName(action) }}
      </NButton>
    </template>
    <template #default>
      {{ disabledTooltip }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";
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
    case "ISSUE_CREATE":
      return t("plan.ready-for-review");
    case "PLAN_CLOSE":
      return t("common.close");
    case "PLAN_REOPEN":
      return t("common.reopen");
  }
};

const actionButtonProps = (action: UnifiedAction) => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
    case "ISSUE_CREATE":
      return {
        type: "primary" as const,
      };
    case "ISSUE_REVIEW_RE_REQUEST":
    case "ISSUE_STATUS_REOPEN":
    case "ISSUE_REVIEW_REJECT":
    case "ISSUE_STATUS_CLOSE":
    case "PLAN_CLOSE":
    case "PLAN_REOPEN":
      return {
        type: "default" as const,
      };
  }
};
</script>
