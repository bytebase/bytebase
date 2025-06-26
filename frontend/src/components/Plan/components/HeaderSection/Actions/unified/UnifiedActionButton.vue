<template>
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

<script setup lang="ts">
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import type { UnifiedAction } from "./types";

defineProps<{
  action: UnifiedAction;
  disabled?: boolean;
}>();

defineEmits<{
  (event: "perform-action", action: UnifiedAction): void;
}>();

const { t } = useI18n();

const actionDisplayName = (action: UnifiedAction): string => {
  switch (action) {
    case "APPROVE":
      return t("common.approve");
    case "REJECT":
      return t("custom-approval.issue-review.send-back");
    case "RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review");
    case "CLOSE":
      return t("issue.batch-transition.close");
    case "REOPEN":
      return t("issue.batch-transition.reopen");
    case "CREATE_ROLLOUT":
      return t("common.create") + " " + t("common.rollout");
    case "CREATE_ISSUE":
      return t("common.create") + " " + t("common.issue");
  }
};

const actionButtonProps = (action: UnifiedAction) => {
  switch (action) {
    case "APPROVE":
    case "CREATE_ROLLOUT":
    case "CREATE_ISSUE":
      return {
        type: "primary" as const,
      };
    case "RE_REQUEST":
    case "REOPEN":
    case "REJECT":
    case "CLOSE":
      return {
        type: "default" as const,
      };
  }
};
</script>
