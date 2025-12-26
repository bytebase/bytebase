<template>
  <NTooltip :disabled="!disabledReason" placement="top">
    <template #trigger>
      <NButton
        :type="action.buttonType"
        :disabled="disabled"
        size="medium"
        tag="div"
        @click="$emit('execute', action.id)"
      >
        {{ action.label(context) }}
      </NButton>
    </template>
    <template #default>
      {{ disabledReason }}
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import type { ActionContext, ActionDefinition, UnifiedAction } from "../types";

defineProps<{
  action: ActionDefinition;
  context: ActionContext;
  disabled?: boolean;
  disabledReason?: string;
}>();

defineEmits<{
  (e: "execute", id: UnifiedAction): void;
}>();
</script>
