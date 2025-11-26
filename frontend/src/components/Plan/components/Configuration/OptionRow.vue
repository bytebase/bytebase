<template>
  <div class="flex items-center justify-between gap-3">
    <!-- Left: Label + Tooltip -->
    <div class="flex items-center gap-1">
      <label class="text-sm text-main whitespace-nowrap">
        <slot name="label" />
      </label>
      <NTooltip v-if="$slots.tooltip">
        <template #trigger>
          <heroicons:information-circle
            class="w-4 h-4 text-control-light cursor-help"
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
    <!-- Right: Value (switch/selector) -->
    <slot />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
</script>
