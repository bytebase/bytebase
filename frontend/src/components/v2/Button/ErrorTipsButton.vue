<template>
  <NTooltip
    :disabled="disabled || (errors ?? []).length === 0"
    v-bind="tooltipProps"
  >
    <template #trigger>
      <NButton
        tag="div"
        :disabled="disabled || (errors ?? []).length > 0"
        v-bind="{
          ...$attrs,
          ...buttonProps,
        }"
        @click="$emit('click', $event)"
      >
        <template #icon>
          <slot name="icon" />
        </template>
        <template #default>
          <slot name="default" />
        </template>
      </NButton>
    </template>
    <template #default>
      <slot name="tooltip" :errors="errors">
        <ErrorList :errors="errors ?? []" :class="errorListClass" />
      </slot>
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NTooltip, NButton, TooltipProps, ButtonProps } from "naive-ui";
import ErrorList from "@/components/misc/ErrorList.vue";
import { VueClass } from "@/utils";

defineProps<{
  disabled?: boolean;
  errors?: string[];
  tooltipProps?: TooltipProps;
  buttonProps?: ButtonProps;
  errorListClass?: VueClass;
}>();

defineEmits<{
  (event: "click", e: MouseEvent): void;
}>();
</script>
