<template>
  <NTooltip :disabled="Object.keys(labels).length <= minimumShowCount">
    <template #trigger>
      <div
        v-if="Object.entries(labels).length > 0"
        class="flex items-center space-x-1"
      >
        <div
          v-for="[key, label] in Object.entries(labels).slice(
            0,
            minimumShowCount
          )"
          :key="key"
          class="rounded-lg bg-gray-100 group-hover:bg-gray-200 py-1 px-2 text-sm"
        >
          {{ `${key}:${label}` }}
        </div>
        <span v-if="Object.keys(labels).length > minimumShowCount">...</span>
      </div>
      <div v-else>{{ placeholder }}</div>
    </template>
    <div class="text-sm flex flex-col space-y-1">
      <div v-for="[key, label] in Object.entries(labels)" :key="key">
        {{ `${key}:${label}` }}
      </div>
    </div>
  </NTooltip>
</template>

<script setup lang="ts">
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

const minimumShowCount = Math.max(1, props.showCount);
</script>
