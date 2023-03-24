<template>
  <NInput
    :value="value"
    :placeholder="$t('custom-approval.security-rule.condition.input-value')"
    :disabled="!allowAdmin"
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { useExprEditorContext } from "../context";

defineProps<{
  value: string;
}>();

defineEmits<{
  (event: "update:value", value: string): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;
</script>
