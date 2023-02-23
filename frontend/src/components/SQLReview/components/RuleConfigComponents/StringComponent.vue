<template>
  <input
    :value="value"
    type="text"
    :disabled="disabled || !editable"
    :class="[
      'shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md',
      (disabled || !editable) && 'cursor-not-allowed',
    ]"
    :placeholder="config.payload.default as string"
    @change="$emit('update:value', ($event.target as HTMLInputElement).value)"
  />
</template>

<script lang="ts" setup>
import { RuleConfigComponent } from "@/types";

defineProps<{
  config: RuleConfigComponent;
  value: string;
  disabled: boolean;
  editable: boolean;
}>();

defineEmits<{
  (event: "update:value", value: string): void;
}>();
</script>
