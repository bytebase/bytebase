<template>
  <div
    v-if="sections.length > 0"
    class="w-full font-mono text-xs bg-gray-50 border border-gray-200 overflow-hidden rounded"
  >
    <div
      v-for="section in sections"
      :key="section.id"
      class="border-b border-gray-200 last:border-b-0"
    >
      <!-- Section Header -->
      <div
        class="flex items-center gap-x-2 px-3 py-1.5 bg-white hover:bg-gray-50 cursor-pointer select-none"
        @click="toggleSection(section.id)"
      >
        <!-- Expand/Collapse Icon -->
        <component
          :is="isSectionExpanded(section.id) ? ChevronDownIcon : ChevronRightIcon"
          class="w-3.5 h-3.5 text-gray-400 shrink-0"
        />
        <!-- Status Icon -->
        <component
          :is="section.statusIcon"
          class="w-3.5 h-3.5 shrink-0"
          :class="[
            section.statusClass,
            { 'animate-spin': section.status === 'running' },
          ]"
        />
        <!-- Section Title -->
        <span class="text-gray-700">{{ section.label }}</span>
        <!-- Entry Count -->
        <span v-if="section.entryCount > 1" class="text-gray-400">
          ({{ section.entryCount }})
        </span>
        <!-- Spacer -->
        <span class="flex-1" />
        <!-- Duration -->
        <span v-if="section.duration" class="text-gray-500 tabular-nums">
          {{ section.duration }}
        </span>
      </div>

      <!-- Section Content with Virtual Scroll -->
      <NVirtualList
        v-if="isSectionExpanded(section.id)"
        :items="section.items"
        :item-size="ITEM_HEIGHT"
        item-resizable
        :style="{ maxHeight: `${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px` }"
        class="bg-gray-50 border-t border-gray-100"
      >
        <template #default="{ item, index }">
          <div
            class="flex items-start gap-x-2 px-3 py-0.5 hover:bg-gray-100"
            :class="{ 'border-t border-gray-100': index > 0 }"
          >
            <!-- Row Number -->
            <span class="text-gray-300 w-6 text-right shrink-0 tabular-nums">
              {{ index + 1 }}
            </span>
            <!-- Timestamp -->
            <span class="text-gray-400 shrink-0 tabular-nums">
              {{ item.time }}
            </span>
            <!-- Relative Time -->
            <span
              v-if="item.relativeTime"
              class="text-gray-300 shrink-0 tabular-nums"
            >
              {{ item.relativeTime }}
            </span>
            <!-- Status Indicator -->
            <span :class="item.levelClass" class="shrink-0">
              {{ item.levelIndicator }}
            </span>
            <!-- Detail -->
            <span :class="item.detailClass" class="break-all">
              {{ item.detail }}
            </span>
            <!-- Affected Rows -->
            <span
              v-if="item.affectedRows !== undefined"
              class="text-gray-400 shrink-0 ml-auto"
            >
              {{ item.affectedRows }} rows
            </span>
          </div>
        </template>
      </NVirtualList>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ChevronDownIcon, ChevronRightIcon } from "lucide-vue-next";
import { NVirtualList } from "naive-ui";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { useTaskRunLogSections } from "./useTaskRunLogSections";

const ITEM_HEIGHT = 20;
const MAX_VISIBLE_ITEMS = 10;

const props = defineProps<{
  entries: TaskRunLogEntry[];
  sheet?: Sheet;
}>();

const { sections, toggleSection, isSectionExpanded } = useTaskRunLogSections(
  () => props.entries,
  () => props.sheet
);
</script>
