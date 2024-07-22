<template>
  <NButton class="group" size="small" @click.stop="handleClick">
    <template #icon>
      <slot name="icon" />
    </template>
    <span class="leading-6" :class="[factor.disabled && 'line-through']">
      {{ readableSQLEditorTreeFactor(factor.factor) }}
    </span>
    <button
      class="hidden group-hover:flex bg-gray-100 absolute rounded-full top-0 right-0 hover:bg-gray-300 z-10 translate-x-[50%] translate-y-[-40%] w-4 h-4 items-center justify-center"
      @click.stop="$emit('remove')"
    >
      <heroicons:x-mark class="w-3.5 h-3.5" />
    </button>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import type { StatefulSQLEditorTreeFactor as StatefulFactor } from "@/types";
import { readableSQLEditorTreeFactor } from "@/types";

defineProps<{
  factor: StatefulFactor;
}>();
const emit = defineEmits<{
  (event: "toggle-disabled"): void;
  (event: "remove"): void;
}>();

const handleClick = () => {
  emit("toggle-disabled");
};
</script>
