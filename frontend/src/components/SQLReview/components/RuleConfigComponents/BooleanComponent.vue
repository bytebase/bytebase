<template>
  <NCheckbox
    :checked="value"
    :disabled="disabled"
    @update:checked="$emit('update:value', $event)"
  >
    <div class="flex items-center gap-x-1">
      <p class="font-medium">
        {{ title }}
      </p>
      <NTooltip v-if="tooltip">
        <template #trigger>
          <CircleHelpIcon class="w-4" />
        </template>
        <span>{{ tooltip }}</span>
      </NTooltip>
    </div>
  </NCheckbox>
</template>

<script lang="ts" setup>
import { CircleHelpIcon } from "lucide-vue-next";
import { NCheckbox, NTooltip } from "naive-ui";
import type { RuleConfigComponent } from "@/types";

defineProps<{
  title: string;
  tooltip?: string;
  config: RuleConfigComponent;
  value: boolean;
  disabled: boolean;
}>();

defineEmits<{
  (event: "update:value", value: boolean): void;
}>();
</script>
