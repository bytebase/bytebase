<template>
  <NInputNumber
    :value="value"
    :disabled="disabled"
    :placeholder="`${config.payload.default}`"
    @update:value="$emit('update:value', $event as number)"
  />
</template>

<script lang="ts" setup>
import { NInputNumber } from "naive-ui";
import { RuleConfigComponent } from "@/types";

defineProps<{
  config: RuleConfigComponent;
  value: number;
  disabled: boolean;
}>();

defineEmits<{
  (event: "update:value", value: number): void;
}>();
</script>
