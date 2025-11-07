<template>
  <NTooltip :disabled="Object.keys(labels).length <= minimumShowCount">
    <template #trigger>
      <div
        v-if="Object.entries(labels).length > 0"
        class="flex items-center gap-x-1"
      >
        <div
          v-for="[key, label] in displayLabels"
          :key="key"
          class="rounded-lg bg-gray-100 group-hover:bg-gray-200 py-0.5 px-2 text-sm"
        >
          {{ `${key}:${label}` }}
        </div>
        <span v-if="Object.keys(labels).length > minimumShowCount">...</span>
      </div>
      <div v-else>{{ placeholder }}</div>
    </template>
    <div class="text-sm flex flex-col gap-y-1">
      <div v-for="[key, label] in Object.entries(labels)" :key="key">
        {{ `${key}:${label}` }}
      </div>
    </div>
  </NTooltip>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    labels: {
      [key: string]: string;
    };
    showCount: number;
    placeholder?: string;
  }>(),
  {
    placeholder: "",
  }
);

const minimumShowCount = computed(() => {
  return Math.max(1, props.showCount);
});

const displayLabels = computed(() => {
  return Object.entries(props.labels).slice(0, minimumShowCount.value);
});
</script>
