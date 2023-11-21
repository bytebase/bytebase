<template>
  <div class="grid">
    <NButton
      v-for="(option, index) in options"
      :key="option.value"
      class="bb-radio-grid--button"
      size="large"
      ghost
      :type="option.value === value ? 'primary' : 'default'"
      :disabled="disabled"
      v-bind="buttonProps ? buttonProps(option.value, index) : undefined"
      @click="$emit('update:value', option.value)"
    >
      <div class="flex flex-row items-center gap-x-2">
        <NRadio
          :checked="value === option.value"
          :disabled="disabled"
          size="large"
          class="pointer-events-none"
          v-bind="radioProps ? radioProps(option.value, index) : undefined"
        />
        <slot
          name="item"
          :option="(option as RadioGridOption<any>)"
          :index="index"
        />
      </div>
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { ButtonProps, NButton, NRadio, RadioProps } from "naive-ui";
import { RadioGridOption } from "./types";

defineProps<{
  value?: string | number | undefined | null;
  options: RadioGridOption<string | number>[];
  disabled?: boolean;
  buttonProps?: (value: string | number, index: number) => ButtonProps;
  radioProps?: (value: string | number, index: number) => RadioProps;
}>();
defineEmits<{
  (event: "update:value", value: string | number): void;
}>();
</script>

<style lang="postcss" scoped>
.bb-radio-grid--button :deep(.n-button__content) {
  @apply w-full justify-start;
}
</style>
