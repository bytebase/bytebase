<template>
  <NInputNumber
    :value="value"
    :show-button="false"
    :placeholder="$t('custom-approval.security-rule.condition.input-value')"
    :disabled="!allowAdmin"
    size="small"
    style="width: auto; max-width: 5rem; overflow-x: hidden"
    @update:value="$emit('update:value', $event ?? 0)"
  />
</template>

<script lang="ts" setup>
import { NInputNumber } from "naive-ui";
import { useExprEditorContext } from "../context";

defineProps<{
  value: number;
}>();

defineEmits<{
  (event: "update:value", value: number): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;
</script>
