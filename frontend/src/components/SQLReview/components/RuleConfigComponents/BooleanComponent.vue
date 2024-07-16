<template>
  <NCheckbox
    :label="title"
    :checked="value"
    :disabled="disabled"
    @update:checked="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import type { RuleConfigComponent } from "@/types";

defineProps<{
  title: string;
  config: RuleConfigComponent;
  value: boolean;
  disabled: boolean;
}>();

defineEmits<{
  (event: "update:value", value: boolean): void;
}>();
</script>
