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
import { usePlanAction } from "./action";
import type { UnifiedAction } from "./types";

defineProps<{
  action: UnifiedAction;
  disabled?: boolean;
  disabledTooltip?: string;
}>();

defineEmits<{
  (event: "perform-action", action: UnifiedAction): void;
}>();

const { actionDisplayName } = usePlanAction();

const actionButtonProps = (action: UnifiedAction) => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
    case "ISSUE_CREATE":
    case "ROLLOUT_START":
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
