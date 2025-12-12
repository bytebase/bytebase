<template>
  <div
    class="flex items-center gap-x-2 py-1.5 bg-white hover:bg-gray-50 cursor-pointer select-none"
    :class="indent ? 'px-6' : 'px-3'"
    @click="$emit('toggle')"
  >
    <component
      :is="isExpanded ? ChevronDownIcon : ChevronRightIcon"
      class="w-3.5 h-3.5 text-gray-400 shrink-0"
    />
    <component
      :is="section.statusIcon"
      class="w-3.5 h-3.5 shrink-0"
      :class="[section.statusClass, { 'animate-spin': section.status === 'running' }]"
    />
    <span class="text-gray-700">{{ section.label }}</span>
    <span v-if="section.entryCount > 1" class="text-gray-400">
      ({{ section.entryCount }})
    </span>
    <span class="flex-1" />
    <span v-if="section.duration" class="text-gray-500 tabular-nums">
      {{ section.duration }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { ChevronDownIcon, ChevronRightIcon } from "lucide-vue-next";
import type { Section } from "./types";

defineProps<{
  section: Section;
  isExpanded: boolean;
  indent?: boolean;
}>();

defineEmits<{
  toggle: [];
}>();
</script>
