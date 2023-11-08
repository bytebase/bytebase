<template>
  <input
    :value="value"
    type="number"
    :disabled="disabled"
    :class="[
      'shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md',
      disabled && 'cursor-not-allowed',
    ]"
    :placeholder="`${config.payload.default}`"
    @change="
      $emit('update:value', ($event.target as HTMLInputElement).valueAsNumber)
    "
  />
</template>

<script lang="ts" setup>
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
