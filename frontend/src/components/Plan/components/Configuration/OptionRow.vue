<template>
  <div class="flex items-center gap-2">
    <!-- Label + Tooltip -->
    <div class="flex items-center gap-1">
      <label class="text-sm text-control-placeholder whitespace-nowrap">
        <slot name="label" />
      </label>
      <NTooltip v-if="$slots.tooltip">
        <template #trigger>
          <heroicons:information-circle
            class="w-3.5 h-3.5 text-control-light cursor-help"
          />
        </template>
        <template #default>
          <div class="max-w-xs">
            <slot name="tooltip" />
          </div>
        </template>
      </NTooltip>
      <slot name="suffix" />
    </div>
    <!-- Value -->
    <slot />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
</script>
